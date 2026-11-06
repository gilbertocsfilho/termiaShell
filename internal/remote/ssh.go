package remote

import (
	"errors"
	"fmt"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"io"
	"net"
	"os"
	"path"
	"sort"
	"sync"
	"termiashell/internal/core"
	"time"
)

type terminal struct {
	reader  *io.PipeReader
	writer  *io.PipeWriter
	input   io.WriteCloser
	session *ssh.Session
	client  *ssh.Client
	once    sync.Once
	done    chan struct{}
}

func (t *terminal) Read(b []byte) (int, error)  { return t.reader.Read(b) }
func (t *terminal) Write(b []byte) (int, error) { return t.input.Write(b) }
func (t *terminal) Resize(c, r int) error       { return t.session.WindowChange(r, c) }
func (t *terminal) Close() error {
	t.once.Do(func() { close(t.done); t.client.Close(); t.reader.Close(); t.writer.Close() })
	return nil
}
func Open(p core.Profile, password, passphrase string, hostCheck ssh.HostKeyCallback) (core.Terminal, core.Files, string, string, error) {
	auth := []ssh.AuthMethod{}
	if p.KeyPath != "" {
		b, e := os.ReadFile(p.KeyPath)
		if e != nil {
			return nil, nil, "", "", e
		}
		var signer ssh.Signer
		if passphrase != "" {
			signer, e = ssh.ParsePrivateKeyWithPassphrase(b, []byte(passphrase))
		} else {
			signer, e = ssh.ParsePrivateKey(b)
		}
		if e != nil {
			return nil, nil, "", "", errors.New("Não foi possível ler a chave privada; confira a passphrase")
		}
		auth = append(auth, ssh.PublicKeys(signer))
	} else {
		auth = append(auth, ssh.Password(password))
	}
	conn, e := net.DialTimeout("tcp", core.Address(p), 15*time.Second)
	if e != nil {
		return nil, nil, "", "", e
	}
	conn.SetDeadline(time.Now().Add(20 * time.Second))
	cc, ch, req, e := ssh.NewClientConn(conn, core.Address(p), &ssh.ClientConfig{User: p.User, Auth: auth, HostKeyCallback: hostCheck, Timeout: 15 * time.Second})
	if e != nil {
		conn.Close()
		return nil, nil, "", "", e
	}
	conn.SetDeadline(time.Time{})
	client := ssh.NewClient(cc, ch, req)
	ses, e := client.NewSession()
	if e != nil {
		client.Close()
		return nil, nil, "", "", e
	}
	if e = ses.RequestPty("xterm-256color", 30, 100, ssh.TerminalModes{ssh.ECHO: 1, ssh.TTY_OP_ISPEED: 14400, ssh.TTY_OP_OSPEED: 14400}); e != nil {
		client.Close()
		return nil, nil, "", "", e
	}
	input, e := ses.StdinPipe()
	if e != nil {
		client.Close()
		return nil, nil, "", "", e
	}
	r, w := io.Pipe()
	ses.Stdout = w
	ses.Stderr = w
	if e = ses.Shell(); e != nil {
		client.Close()
		r.Close()
		w.Close()
		return nil, nil, "", "", e
	}
	t := &terminal{reader: r, writer: w, input: input, session: ses, client: client, done: make(chan struct{})}
	go func() { e := ses.Wait(); w.CloseWithError(e) }()
	go func() {
		ticker := time.NewTicker(20 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-t.done:
				return
			case <-ticker.C:
				result := make(chan error, 1)
				go func() { _, _, e := client.SendRequest("keepalive@openssh.com", true, nil); result <- e }()
				select {
				case e := <-result:
					if e != nil {
						t.Close()
						return
					}
				case <-time.After(10 * time.Second):
					t.Close()
					return
				case <-t.done:
					return
				}
			}
		}
	}()
	type result struct {
		c *sftp.Client
		e error
	}
	ready := make(chan result, 1)
	go func() { s, e := sftp.NewClient(client); ready <- result{s, e} }()
	var fs *sftp.Client
	var warn string
	select {
	case res := <-ready:
		fs = res.c
		if res.e != nil {
			warn = "Terminal conectado; SFTP indisponível"
		}
	case <-time.After(10 * time.Second):
		t.Close()
		return nil, nil, "", "", errors.New("Timeout iniciando SFTP")
	}
	dir := p.Directory
	if fs == nil {
		return t, nil, "/", warn, nil
	}
	if dir == "" {
		dir, e = fs.Getwd()
		if e != nil {
			dir = "/"
		}
	}
	return t, &Files{fs}, dir, warn, nil
}

type Files struct{ c *sftp.Client }

func (f *Files) check(p string) error {
	if e := core.ValidateFilePath(p); e != nil {
		return e
	}
	for cur := path.Clean(p); ; cur = path.Dir(cur) {
		st, e := f.c.Lstat(cur)
		if e == nil && st.Mode()&os.ModeSymlink != 0 {
			return errors.New("Links simbólicos não são seguidos")
		}
		if cur == "/" {
			break
		}
	}
	return nil
}
func (f *Files) List(p string) ([]core.Entry, error) {
	if e := f.check(p); e != nil {
		return nil, e
	}
	ls, e := f.c.ReadDir(p)
	if e != nil {
		return nil, e
	}
	out := []core.Entry{}
	for _, s := range ls {
		out = append(out, core.Entry{Name: s.Name(), Directory: s.IsDir(), Link: s.Mode()&os.ModeSymlink != 0, Size: s.Size(), Modified: s.ModTime(), Mode: fmt.Sprintf("%04o", s.Mode().Perm())})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Directory != out[j].Directory {
			return out[i].Directory
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}
func (f *Files) Open(p string) (io.ReadCloser, error) {
	if e := f.check(p); e != nil {
		return nil, e
	}
	s, e := f.c.Stat(p)
	if e != nil {
		return nil, e
	}
	if !s.Mode().IsRegular() {
		return nil, errors.New("Selecione um arquivo regular")
	}
	return f.c.Open(p)
}
func (f *Files) Create(p string) (io.WriteCloser, error) {
	if e := f.check(p); e != nil {
		return nil, e
	}
	return f.c.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL)
}
func (f *Files) Mkdir(p string) error {
	if e := f.check(p); e != nil {
		return e
	}
	return f.c.Mkdir(p)
}
func (f *Files) Rename(a, b string) error {
	if e := f.check(a); e != nil {
		return e
	}
	if e := f.check(b); e != nil {
		return e
	}
	exists, e := f.Exists(b)
	if e != nil {
		return e
	}
	if exists {
		return errors.New("Destino já existe")
	}
	return f.c.Rename(a, b)
}
func (f *Files) Chmod(p string, mode os.FileMode) error {
	if e := f.check(p); e != nil {
		return e
	}
	return f.c.Chmod(p, mode)
}
func (f *Files) Remove(p string) error {
	if e := f.check(p); e != nil {
		return e
	}
	s, e := f.c.Lstat(p)
	if e != nil {
		return e
	}
	if s.IsDir() {
		return f.c.RemoveDirectory(p)
	}
	return f.c.Remove(p)
}
func (f *Files) Exists(p string) (bool, error) {
	if e := f.check(p); e != nil {
		return false, e
	}
	_, e := f.c.Lstat(p)
	if errors.Is(e, os.ErrNotExist) {
		return false, nil
	}
	return e == nil, e
}
func (f *Files) Close() error { return f.c.Close() }
func FingerprintError(key ssh.PublicKey) error {
	return fmt.Errorf("HOSTKEY|%s|%s", ssh.FingerprintSHA256(key), key.Type())
}
