package app

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"termiashell/internal/core"
	"testing"
)

func TestSecurityCredentialsNeverPersist(t *testing.T) {
	a, e := New(t.TempDir(), &testUI{})
	if e != nil {
		t.Fatal(e)
	}
	defer a.Shutdown()
	var req ConnectRequest
	raw := `{"profile":{"name":"Servidor","kind":"ssh","host":"example.test","port":22,"user":"tester","password":"nested-secret"},"password":"password-secret","passphrase":"passphrase-secret"}`
	if e = json.Unmarshal([]byte(raw), &req); e != nil {
		t.Fatal(e)
	}
	if _, e = a.SaveProfile(req.Profile); e != nil {
		t.Fatal(e)
	}
	entries, _ := os.ReadDir(a.data)
	for _, entry := range entries {
		b, e := os.ReadFile(filepath.Join(a.data, entry.Name()))
		if e != nil {
			t.Fatal(e)
		}
		for _, secret := range []string{"nested-secret", "password-secret", "passphrase-secret"} {
			if bytes.Contains(b, []byte(secret)) {
				t.Fatal("segredo persistido", entry.Name())
			}
		}
	}
}
func TestSecuritySessionAndInputLimits(t *testing.T) {
	a, e := New(t.TempDir(), &testUI{})
	if e != nil {
		t.Fatal(e)
	}
	defer a.Shutdown()
	if a.SendInput("missing", "test") == nil {
		t.Fatal("sessão inexistente aceita")
	}
	a.sessions["closed"] = &session{term: &fakeTerm{}, closed: make(chan struct{})}
	a.sessions["closed"].close()
	if a.SendInput("closed", "test") == nil {
		t.Fatal("sessão encerrada aceita")
	}
	term := &limitedTerminal{}
	files := &limitedFiles{}
	a.sessions["live"] = &session{term: term, files: files, closed: make(chan struct{})}
	if e = a.SendInput("live", "legítimo"); e != nil || term.writes != 1 {
		t.Fatal("baseline de entrada falhou", e)
	}
	if a.SendInput("live", strings.Repeat("x", 256*1024+1)) == nil {
		t.Fatal("entrada ilimitada")
	}
	for _, dims := range [][2]int{{0, 20}, {80, 0}, {1001, 30}, {80, 501}, {-1, 10}} {
		if a.ResizeSession("live", dims[0], dims[1]) == nil {
			t.Fatal("dimensões ilimitadas")
		}
	}
	for _, p := range []string{"/", "/../", "/link/../"} {
		if a.Remove("live", p) == nil {
			t.Fatal("raiz/caminho ambíguo aceito")
		}
	}
	if term.writes != 1 || term.resizes != 0 || files.removals != 0 {
		t.Fatal("entrada rejeitada chegou ao adaptador")
	}
}

type limitedTerminal struct {
	fakeTerm
	writes, resizes int
}

func (t *limitedTerminal) Write(b []byte) (int, error) { t.writes++; return len(b), nil }
func (t *limitedTerminal) Resize(int, int) error       { t.resizes++; return nil }

type limitedFiles struct {
	core.Files
	removals int
}

func (f *limitedFiles) Close() error        { return nil }
func (f *limitedFiles) Remove(string) error { f.removals++; return nil }

type hostileListing struct {
	core.Files
	name string
}

func (f *hostileListing) List(string) ([]core.Entry, error) { return []core.Entry{{Name: f.name}}, nil }
func (f *hostileListing) Close() error                      { return nil }
func TestSecurityHostileRemoteNamesRejected(t *testing.T) {
	a, e := New(t.TempDir(), &testUI{})
	if e != nil {
		t.Fatal(e)
	}
	defer a.Shutdown()
	fs := &hostileListing{}
	a.sessions["hostile"] = &session{term: &fakeTerm{}, files: fs, closed: make(chan struct{})}
	for _, name := range []string{"..", "../victim", "folder/file", "evil\x00file", "evil\nfile"} {
		fs.name = name
		if _, e = a.ListFiles("hostile", "/home"); e == nil {
			t.Fatal("nome remoto aceito", name)
		}
	}
}
