package platform

import (
	"errors"
	"github.com/creack/pty"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"termiashell/internal/core"
	"termiashell/internal/localfiles"
)

func Capabilities() core.Capabilities {
	return core.Capabilities{OS: "linux", LocalKind: "local"}
}
func DefaultTarget() (string, error) { return "local", nil }
func Targets() ([]core.Target, error) {
	return []core.Target{{ID: "local", Name: "Terminal local", Kind: "local"}}, nil
}

type terminal struct {
	*os.File
	cmd *exec.Cmd
}

func (t *terminal) Resize(c, r int) error {
	return pty.Setsize(t.File, &pty.Winsize{Cols: uint16(c), Rows: uint16(r)})
}
func (t *terminal) Close() error {
	if t.cmd.Process != nil {
		syscall.Kill(-t.cmd.Process.Pid, syscall.SIGTERM)
	}
	return t.File.Close()
}
func OpenLocal(p core.Profile) (core.Terminal, core.Files, string, error) {
	if p.Kind != "local" {
		return nil, nil, "", errors.New("WSL só está disponível no Windows")
	}
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	dir := p.Directory
	if dir == "" {
		dir, _ = os.UserHomeDir()
	}
	if !filepath.IsAbs(dir) {
		return nil, nil, "", errors.New("Diretório deve ser absoluto")
	}
	c := exec.Command(shell)
	c.Dir = dir
	c.Env = append(os.Environ(), "TERM=xterm-256color")
	f, e := pty.StartWithSize(c, &pty.Winsize{Cols: 100, Rows: 30})
	if e != nil {
		return nil, nil, "", e
	}
	go c.Wait()
	files := &localfiles.Files{Resolve: func(v string) (string, error) {
		if e := core.ValidateFilePath(v); e != nil {
			return "", e
		}
		return filepath.Clean(v), nil
	}}
	return &terminal{f, c}, files, dir, nil
}
func Lock(dir string) (func(), error) {
	f, e := os.OpenFile(filepath.Join(dir, "instance.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if e != nil {
		return nil, e
	}
	if e = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); e != nil {
		f.Close()
		return nil, errors.New("Pasta de dados em uso")
	}
	return func() { f.Close() }, nil
}
