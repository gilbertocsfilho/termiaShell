package app

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path"
	"path/filepath"
	"sync"
	"termiashell/internal/core"
	"termiashell/internal/platform"
	"termiashell/internal/remote"
	"termiashell/internal/store"
	"time"
)

type UI interface {
	Emit(string, ...interface{})
	OpenFile() (string, error)
	OpenCSV() (string, error)
	SaveFile(string) (string, error)
}
type session struct {
	id       string
	term     core.Terminal
	files    core.Files
	closed   chan struct{}
	once     sync.Once
	readOnce sync.Once
	ack      chan struct{}
	writeMu  sync.Mutex
}

func (s *session) close() {
	s.once.Do(func() {
		close(s.closed)
		s.term.Close()
		if s.files != nil {
			s.files.Close()
		}
	})
}

type App struct {
	data      string
	store     *store.Store
	ui        UI
	mu        sync.Mutex
	csvMu     sync.Mutex
	sessions  map[string]*session
	pending   map[string]pendingTrust
	trustMu   sync.Mutex
	transfers map[string]context.CancelFunc
	slots     chan struct{}
	ctx       context.Context
	cancel    context.CancelFunc
}

func New(data string, ui UI) (*App, error) {
	if e := os.MkdirAll(data, 0700); e != nil {
		return nil, e
	}
	f, e := os.OpenFile(filepath.Join(data, "known_hosts"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if e != nil {
		return nil, e
	}
	f.Close()
	ctx, cancel := context.WithCancel(context.Background())
	return &App{data: data, ui: ui, store: &store.Store{Dir: data}, sessions: map[string]*session{}, pending: map[string]pendingTrust{}, transfers: map[string]context.CancelFunc{}, slots: make(chan struct{}, 2), ctx: ctx, cancel: cancel}, nil
}

func id() string {
	b := make([]byte, 16)
	if _, e := rand.Read(b); e != nil {
		panic(e)
	}
	return hex.EncodeToString(b)
}
func (a *App) GetCapabilities() core.Capabilities { return platform.Capabilities() }

func (a *App) ListLocalTargets() ([]core.Target, error) { return platform.Targets() }
func (a *App) GetDefaultLocalTarget() (string, error)   { return platform.DefaultTarget() }
func (a *App) ListProfiles() ([]core.Profile, error)    { return a.store.Load() }
func (a *App) SaveProfile(p core.Profile) (core.Profile, error) {
	if p.Kind != "ssh" {
		return core.Profile{}, errors.New("WSL e terminal local conectam direto e não são salvos como perfil")
	}
	if p.ID == "" {
		p.ID = id()
	}
	return p, a.store.Save(p)
}
func (a *App) DeleteProfile(profileID string) error { return a.store.Delete(profileID) }

type ConnectRequest struct {
	Profile    core.Profile `json:"profile"`
	Password   string       `json:"password"`
	Passphrase string       `json:"passphrase"`
}
type Opened struct {
	ID        string `json:"id"`
	Directory string `json:"directory"`
	Warning   string `json:"warning"`
	HasFiles  bool   `json:"hasFiles"`
}

func (a *App) OpenSession(req ConnectRequest) (Opened, error) {
	p := req.Profile
	if e := p.Validate(); e != nil {
		return Opened{}, e
	}
	var t core.Terminal
	var fs core.Files
	var dir, warn string
	var e error
	if p.Kind == "ssh" {
		t, fs, dir, warn, e = remote.Open(p, req.Password, req.Passphrase, a.hostKeyCallback(p))
	} else if p.Kind == "wsl" || p.Kind == "local" {
		t, fs, dir, e = platform.OpenLocal(p)
	} else {
		return Opened{}, errors.New("Tipo de sessão não suportado")
	}
	if e != nil {
		return Opened{}, e
	}
	s := &session{id: id(), term: t, files: fs, closed: make(chan struct{}), ack: make(chan struct{}, 1)}
	a.mu.Lock()
	a.sessions[s.id] = s
	a.mu.Unlock()
	return Opened{s.id, dir, warn, fs != nil}, nil
}
func (a *App) get(sid string) (*session, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	s := a.sessions[sid]
	if s == nil {
		return nil, errors.New("Sessão encerrada")
	}
	select {
	case <-s.closed:
		return nil, errors.New("Sessão encerrada")
	default:
	}
	return s, nil
}
func (a *App) StartReading(sid string) error {
	s, e := a.get(sid)
	if e != nil {
		return e
	}
	s.readOnce.Do(func() {
		go func() {
			defer func() { s.close(); a.ui.Emit("session:closed", sid) }()
			b := make([]byte, 32768)
			for {
				n, err := s.term.Read(b)
				if n > 0 {
					a.ui.Emit("terminal:data", map[string]interface{}{"id": sid, "data": base64.StdEncoding.EncodeToString(b[:n])})
					select {
					case <-s.ack:
					case <-s.closed:
						return
					case <-time.After(60 * time.Second):
						a.ui.Emit("session:error", map[string]string{"id": sid, "message": "Terminal sem resposta; sessão encerrada"})
						return
					}
				}
				if err != nil {
					return
				}
			}
		}()
	})
	return nil
}
func (a *App) Ack(sid string) {
	s, e := a.get(sid)
	if e == nil {
		select {
		case s.ack <- struct{}{}:
		default:
		}
	}
}
func (a *App) SendInput(sid, data string) error {
	if len(data) > 256*1024 {
		return errors.New("Entrada muito grande")
	}
	s, e := a.get(sid)
	if e != nil {
		return e
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	_, e = s.term.Write([]byte(data))
	return e
}
func (a *App) ResizeSession(sid string, c, r int) error {
	if c < 2 || r < 2 || c > 1000 || r > 500 {
		return errors.New("Dimensões inválidas")
	}
	s, e := a.get(sid)
	if e != nil {
		return e
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	return s.term.Resize(c, r)
}
func (a *App) CloseSession(sid string) error {
	a.mu.Lock()
	s := a.sessions[sid]
	delete(a.sessions, sid)
	a.mu.Unlock()
	if s != nil {
		s.close()
	}
	return nil
}
func (a *App) fileSession(sid string) (core.Files, error) {
	s, e := a.get(sid)
	if e != nil {
		return nil, e
	}
	if s.files == nil {
		return nil, errors.New("Arquivos indisponíveis para esta sessão")
	}
	return s.files, nil
}
func validPath(p string) error { return core.ValidateFilePath(p) }
func (a *App) ListFiles(sid, p string) ([]core.Entry, error) {
	if e := validPath(p); e != nil {
		return nil, e
	}
	fs, e := a.fileSession(sid)
	if e != nil {
		return nil, e
	}
	entries, e := fs.List(p)
	if e != nil {
		return nil, e
	}
	for _, entry := range entries {
		if e = core.ValidateEntryName(entry.Name); e != nil {
			return nil, e
		}
	}
	return entries, nil
}
func (a *App) Mkdir(sid, p string) error {
	if e := validPath(p); e != nil {
		return e
	}
	fs, e := a.fileSession(sid)
	if e != nil {
		return e
	}
	return fs.Mkdir(p)
}
func (a *App) Rename(sid, from, to string) error {
	if e := validPath(from); e != nil {
		return e
	}
	if e := validPath(to); e != nil {
		return e
	}
	fs, e := a.fileSession(sid)
	if e != nil {
		return e
	}
	return fs.Rename(from, to)
}
func (a *App) Chmod(sid, p string, mode uint32) error {
	if e := validPath(p); e != nil {
		return e
	}
	if mode > 0777 {
		return errors.New("Permissões inválidas")
	}
	fs, e := a.fileSession(sid)
	if e != nil {
		return e
	}
	return fs.Chmod(p, os.FileMode(mode))
}
func (a *App) Remove(sid, p string) error {
	if e := validPath(p); e != nil {
		return e
	}
	if path.Clean(p) == "/" {
		return errors.New("Não é permitido excluir a raiz")
	}
	fs, e := a.fileSession(sid)
	if e != nil {
		return e
	}
	return fs.Remove(p)
}
func (a *App) HasActiveWork() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, s := range a.sessions {
		select {
		case <-s.closed:
		default:
			return true
		}
	}
	return len(a.transfers) > 0
}
func (a *App) Shutdown() {
	a.cancel()
	a.mu.Lock()
	ss := []*session{}
	for _, s := range a.sessions {
		ss = append(ss, s)
	}
	for _, cancel := range a.transfers {
		cancel()
	}
	a.mu.Unlock()
	for _, s := range ss {
		s.close()
	}
}

// Transferências nunca sobrescrevem arquivos existentes: o usuário escolhe outro nome.
func (a *App) StartTransfer(sid, remotePath, direction string) (string, error) {
	if e := validPath(remotePath); e != nil {
		return "", e
	}
	fs, e := a.fileSession(sid)
	if e != nil {
		return "", e
	}
	var local string
	if direction == "upload" {
		local, e = a.ui.OpenFile()
	} else if direction == "download" {
		local, e = a.ui.SaveFile(path.Base(remotePath))
	} else {
		return "", errors.New("Direção inválida")
	}
	if e != nil || local == "" {
		return "", e
	}
	tid := id()
	ctx, cancel := context.WithCancel(a.ctx)
	a.mu.Lock()
	a.transfers[tid] = cancel
	a.mu.Unlock()
	go a.transfer(ctx, cancel, tid, fs, remotePath, local, direction)
	return tid, nil
}
func (a *App) CancelTransfer(tid string) {
	a.mu.Lock()
	cancel := a.transfers[tid]
	a.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}
func (a *App) transfer(ctx context.Context, cancel context.CancelFunc, tid string, fs core.Files, rp, lp, direction string) {
	var result error
	var copied int64
	defer func() {
		cancel()
		a.mu.Lock()
		delete(a.transfers, tid)
		a.mu.Unlock()
		msg := "Concluída"
		state := "done"
		if result != nil {
			state = "failed"
			msg = result.Error()
		}
		a.ui.Emit("transfer:update", map[string]interface{}{"id": tid, "bytes": copied, "state": state, "message": msg})
	}()
	select {
	case a.slots <- struct{}{}:
		defer func() { <-a.slots }()
	case <-ctx.Done():
		result = ctx.Err()
		return
	}
	var src io.ReadCloser
	var dst io.WriteCloser
	var tmp string
	var size int64
	if direction == "upload" {
		st, e := os.Stat(lp)
		if e != nil {
			result = e
			return
		}
		if !st.Mode().IsRegular() {
			result = errors.New("Selecione arquivo regular")
			return
		}
		size = st.Size()
		rp = path.Join(rp, filepath.Base(lp))
		exists, e := fs.Exists(rp)
		if e != nil {
			result = e
			return
		}
		if exists {
			result = errors.New("Destino já existe; renomeie o arquivo antes de transferir")
			return
		}
		src, result = os.Open(lp)
		if result != nil {
			return
		}
		defer src.Close()
		tmp = path.Join(path.Dir(rp), ".termiashell-"+tid+".part")
		dst, result = fs.Create(tmp)
		if result != nil {
			return
		}
		defer func() {
			if result != nil {
				fs.Remove(tmp)
			}
		}()
	} else {
		if _, e := os.Lstat(lp); !errors.Is(e, os.ErrNotExist) {
			result = errors.New("Destino já existe ou não pode ser verificado; escolha outro nome")
			return
		}
		src, result = fs.Open(rp)
		if result != nil {
			return
		}
		defer src.Close()
		tmp = filepath.Join(filepath.Dir(lp), ".termiashell-"+tid+".part")
		dst, result = os.OpenFile(tmp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if result != nil {
			return
		}
		defer func() {
			if result != nil {
				os.Remove(tmp)
			}
		}()
	}
	// Fechar os streams desbloqueia I/O quando o usuário cancela.
	finished := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			src.Close()
			dst.Close()
		case <-finished:
		}
	}()
	defer close(finished)
	b := make([]byte, 65536)
	last := time.Now()
	for {
		if e := ctx.Err(); e != nil {
			result = e
			break
		}
		n, err := src.Read(b)
		if n > 0 {
			written, e := dst.Write(b[:n])
			copied += int64(written)
			if e != nil {
				result = e
				break
			}
			if written != n {
				result = io.ErrShortWrite
				break
			}
			if time.Since(last) > 150*time.Millisecond {
				a.ui.Emit("transfer:update", map[string]interface{}{"id": tid, "bytes": copied, "total": size, "state": "running", "message": path.Base(rp)})
				last = time.Now()
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			result = err
			break
		}
	}
	if e := dst.Close(); result == nil {
		result = e
	}
	if result != nil {
		return
	}
	if e := ctx.Err(); e != nil {
		result = e
		return
	}
	if direction == "upload" {
		result = fs.Rename(tmp, rp)
	} else {
		if _, e := os.Lstat(lp); !errors.Is(e, os.ErrNotExist) {
			result = errors.New("Destino surgiu durante a transferência")
			return
		}
		result = os.Link(tmp, lp)
		if result == nil {
			result = os.Remove(tmp)
		}
	}
}
