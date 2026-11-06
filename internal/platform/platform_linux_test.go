package platform

import (
	"io"
	"strings"
	"termiashell/internal/core"
	"testing"
	"time"
)

func TestLinuxLocalPTY(t *testing.T) {
	term, files, dir, e := OpenLocal(core.Profile{Name: "local", Kind: "local"})
	if e != nil {
		t.Fatal(e)
	}
	defer files.Close()
	defer term.Close()
	if e = term.Resize(113, 36); e != nil {
		t.Fatal(e)
	}
	done := make(chan string, 1)
	go func() {
		var out strings.Builder
		b := make([]byte, 8192)
		for {
			n, e := term.Read(b)
			out.Write(b[:n])
			if strings.Contains(out.String(), "LINUX_OK_36 113") {
				done <- out.String()
				return
			}
			if e != nil {
				done <- out.String()
				return
			}
		}
	}()
	term.Write([]byte("printf 'LINUX_%s_' OK; stty size\r"))
	select {
	case out := <-done:
		if !strings.Contains(out, "LINUX_OK_36 113") {
			t.Fatal(out)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("PTY timeout")
	}
	go io.Copy(io.Discard, term)
	if _, e = files.List(dir); e != nil {
		t.Fatal(e)
	}
}

func TestSecurityLinuxRejectsWSLProfile(t *testing.T) {
	if _, _, _, e := OpenLocal(core.Profile{Name: "WSL", Kind: "wsl", Target: "Ubuntu"}); e == nil {
		t.Fatal("WSL aceito no Linux")
	}
}
