package platform

import (
	"encoding/binary"
	"io"
	"os"
	"strings"
	"termiashell/internal/core"
	"testing"
	"time"
	"unicode/utf16"
)

func TestWSLDecode(t *testing.T) {
	b := []byte{255, 254}
	for _, u := range utf16.Encode([]rune("Ubuntu\r\nDistribuição\r\n")) {
		b = binary.LittleEndian.AppendUint16(b, u)
	}
	if decodeWSL(b) != "Ubuntu\r\nDistribuição\r\n" {
		t.Fatal(decodeWSL(b))
	}
}

func TestSecurityWindowsRejectsLinuxLocalProfile(t *testing.T) {
	if _, _, _, e := OpenLocal(core.Profile{Name: "local", Kind: "local"}); e == nil {
		t.Fatal("perfil Linux aceito no Windows")
	}
}
func TestWSLCloseWithoutReader(t *testing.T) {
	// Bounded child: Close must stop the owned WSL process tree even without a reader.
	target := os.Getenv("TERMIASHELL_WSL_TEST")
	if target == "" {
		t.Skip("TERMIASHELL_WSL_TEST não definido")
	}
	term, files, _, e := OpenLocal(core.Profile{Name: "close", Kind: "wsl", Target: target})
	if e != nil {
		t.Fatal(e)
	}
	defer files.Close()
	term.Write([]byte("sleep 60\r"))
	done := make(chan struct{})
	go func() { term.Close(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Close bloqueou sem consumidor")
	}
}
func TestWSLReal(t *testing.T) {
	target := os.Getenv("TERMIASHELL_WSL_TEST")
	if target == "" {
		t.Skip("TERMIASHELL_WSL_TEST não definido")
	}
	term, files, dir, e := OpenLocal(core.Profile{Name: "test", Kind: "wsl", Target: target})
	if e != nil {
		t.Fatal(e)
	}
	defer files.Close()
	done := make(chan string, 1)
	go func() {
		var out strings.Builder
		b := make([]byte, 8192)
		for {
			n, e := term.Read(b)
			out.Write(b[:n])
			if strings.Contains(out.String(), "TERMIASHELL_OK_42") {
				done <- out.String()
				return
			}
			if e != nil {
				done <- out.String()
				return
			}
		}
	}()
	if e = term.Resize(112, 35); e != nil {
		t.Fatal(e)
	}
	term.Write([]byte("printf 'TERMIASHELL_%s_%s\\n' OK 42\r"))
	select {
	case out := <-done:
		if !strings.Contains(out, "TERMIASHELL_OK_42") {
			t.Fatal(out)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("terminal output timeout")
	}
	go io.Copy(io.Discard, term)
	closed := make(chan struct{})
	go func() { term.Close(); close(closed) }()
	select {
	case <-closed:
	case <-time.After(5 * time.Second):
		t.Fatal("ConPTY close timeout")
	}
	t.Log("HOME", dir)
	root := "/tmp/termiashell-test-" + time.Now().Format("20060102150405")
	if e = files.Mkdir(root); e != nil {
		t.Fatal(e)
	}
	defer files.Remove(root)
	w, e := files.Create(root + "/unicode.txt")
	if e != nil {
		t.Fatal(e)
	}
	defer files.Remove(root + "/renamed.txt")
	io.WriteString(w, "São Paulo ✓")
	w.Close()
	if e = files.Rename(root+"/unicode.txt", root+"/renamed.txt"); e != nil {
		t.Fatal(e)
	}
	r, e := files.Open(root + "/renamed.txt")
	if e != nil {
		t.Fatal(e)
	}
	b, e := io.ReadAll(r)
	r.Close()
	if e != nil || string(b) != "São Paulo ✓" {
		t.Fatalf("%q %v", b, e)
	}
	ls, e := files.List(root)
	if e != nil || len(ls) != 1 {
		t.Fatalf("%v %v", ls, e)
	}
}
