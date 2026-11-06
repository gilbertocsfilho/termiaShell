package platform

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"github.com/UserExistsError/conpty"
	"golang.org/x/sys/windows"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"termiashell/internal/core"
	"termiashell/internal/localfiles"
	"time"
	"unicode/utf16"
)

func systemExe(name string) string {
	d, e := windows.GetSystemDirectory()
	if e != nil {
		return filepath.Join(os.Getenv("SystemRoot"), "System32", name)
	}
	return filepath.Join(d, name)
}
func Capabilities() core.Capabilities { return core.Capabilities{OS: "windows", LocalKind: "wsl"} }
func DefaultTarget() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, systemExe("wsl.exe"), "--exec", "printenv", "WSL_DISTRO_NAME")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	if err != nil {
		return "", errors.New("Não foi possível encontrar a distribuição WSL padrão")
	}
	target := strings.TrimSpace(string(out))
	if target == "" {
		return "", errors.New("WSL não informou a distribuição padrão")
	}
	return target, nil
}
func Targets() ([]core.Target, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	c := exec.CommandContext(ctx, systemExe("wsl.exe"), "--list", "--quiet")
	c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	b, e := c.Output()
	if e != nil {
		return nil, errors.New("Não foi possível listar WSL. Verifique acesso e distribuições existentes")
	}
	s := decodeWSL(b)
	out := []core.Target{}
	for _, v := range strings.Split(s, "\n") {
		v = strings.TrimSpace(v)
		if v != "" {
			out = append(out, core.Target{ID: v, Name: v, Kind: "wsl"})
		}
	}
	return out, nil
}
func decodeWSL(b []byte) string {
	if len(b) > 1 && (bytes.IndexByte(b, 0) >= 0 || b[0] == 255 && b[1] == 254) {
		u := []uint16{}
		for i := 0; i+1 < len(b); i += 2 {
			u = append(u, binary.LittleEndian.Uint16(b[i:]))
		}
		return strings.TrimPrefix(string(utf16.Decode(u)), "\ufeff")
	}
	return string(b)
}

type terminal struct {
	*conpty.ConPty
	once     sync.Once
	closeErr error
}

func (t *terminal) Close() error {
	t.once.Do(func() {
		// WSL processes can outlive a closed tab. Terminate only this session tree.
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, systemExe("taskkill.exe"), "/PID", strconv.Itoa(t.Pid()), "/T", "/F")
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		_ = cmd.Run() // The process may already have exited normally.
		t.closeErr = t.ConPty.Close()
	})
	return t.closeErr
}

func (t *terminal) Resize(c, r int) error { return t.ConPty.Resize(c, r) }
func OpenLocal(p core.Profile) (core.Terminal, core.Files, string, error) {
	if p.Kind != "wsl" {
		return nil, nil, "", errors.New("Perfil não disponível no Windows")
	}
	targets, e := Targets()
	if e != nil {
		return nil, nil, "", e
	}
	found := false
	for _, t := range targets {
		if t.ID == p.Target {
			found = true
		}
	}
	if !found {
		return nil, nil, "", errors.New("Distribuição WSL não encontrada")
	}
	args := []string{systemExe("wsl.exe"), "--distribution", p.Target}
	if p.User != "" {
		args = append(args, "--user", p.User)
	}
	dir := p.Directory
	if dir == "" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		a := append(append([]string{}, args[1:]...), "--exec", "printenv", "HOME")
		c := exec.CommandContext(ctx, args[0], a...)
		c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		b, err := c.Output()
		if err != nil {
			return nil, nil, "", errors.New("Não foi possível descobrir HOME no WSL")
		}
		dir = strings.TrimSpace(string(b))
	}
	if !path.IsAbs(dir) {
		return nil, nil, "", errors.New("Diretório WSL deve ser absoluto")
	}
	args = append(args, "--cd", dir)
	quoted := []string{}
	for _, a := range args {
		quoted = append(quoted, syscall.EscapeArg(a))
	}
	pty, e := conpty.Start(strings.Join(quoted, " "), conpty.ConPtyDimensions(100, 30))
	if e != nil {
		return nil, nil, "", e
	}
	root := `\\wsl.localhost\` + p.Target
	if _, e := os.Stat(root + `\`); e != nil {
		root = `\\wsl$\` + p.Target
	}
	files := &localfiles.Files{Resolve: func(v string) (string, error) {
		if e := core.ValidateFilePath(v); e != nil {
			return "", e
		}
		if strings.ContainsRune(v, ':') {
			return "", errors.New("Caminho WSL inválido")
		}
		return root + strings.ReplaceAll(path.Clean(v), "/", `\`), nil
	}}
	return &terminal{ConPty: pty}, files, dir, nil
}
func Lock(dir string) (func(), error) {
	p, e := windows.UTF16PtrFromString(filepath.Join(dir, "instance.lock"))
	if e != nil {
		return nil, e
	}
	h, e := windows.CreateFile(p, windows.GENERIC_READ|windows.GENERIC_WRITE, 0, nil, windows.OPEN_ALWAYS, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if e != nil {
		return nil, errors.New("Pasta de dados em uso ou sem permissão")
	}
	return func() { windows.CloseHandle(h) }, nil
}
