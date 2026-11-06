package store

import (
	"os"
	"path/filepath"
	"termiashell/internal/core"
	"testing"
)

func TestStorePreservesCorruptDocument(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "profiles.json")
	os.WriteFile(p, []byte("broken"), 0600)
	s := Store{Dir: d}
	if s.Save(core.Profile{ID: "one", Name: "one", Kind: "ssh", User: "tester", Host: "host", Port: 22}) == nil {
		t.Fatal("corrupt document overwritten")
	}
	b, _ := os.ReadFile(p)
	if string(b) != "broken" {
		t.Fatal("document changed")
	}
}
func TestStoreRoundTrip(t *testing.T) {
	s := Store{Dir: t.TempDir()}
	p := core.Profile{ID: "one", Name: "Servidor São Paulo", Kind: "ssh", User: "tester", Host: "localhost", Port: 22, KeyPath: "/key"}
	if e := s.Save(p); e != nil {
		t.Fatal(e)
	}
	p.Name = "Atualizado"
	if e := s.Save(p); e != nil {
		t.Fatal(e)
	}
	ps, e := s.Load()
	if e != nil || len(ps) != 1 || ps[0].Name != p.Name {
		t.Fatalf("%v %v", ps, e)
	}
	if e = s.Delete("one"); e != nil {
		t.Fatal(e)
	}
	ps, _ = s.Load()
	if len(ps) != 0 {
		t.Fatal(ps)
	}
}
