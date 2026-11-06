package localfiles

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestFilesNoClobberAndEmptyDirectory(t *testing.T) {
	dir := t.TempDir()
	f := &Files{Resolve: func(p string) (string, error) { return filepath.Join(dir, p), nil }}
	if e := f.Mkdir("sub"); e != nil {
		t.Fatal(e)
	}
	w, e := f.Create("sub/a.txt")
	if e != nil {
		t.Fatal(e)
	}
	io.WriteString(w, "conteúdo Unicode ✓")
	w.Close()
	if _, e = f.Create("sub/a.txt"); e == nil {
		t.Fatal("overwrote file")
	}
	if e = f.Remove("sub"); e == nil {
		t.Fatal("removed nonempty directory")
	}
	if e = f.Rename("sub/a.txt", "sub/b.txt"); e != nil {
		t.Fatal(e)
	}
	r, e := f.Open("sub/b.txt")
	if e != nil {
		t.Fatal(e)
	}
	b, _ := io.ReadAll(r)
	r.Close()
	if string(b) != "conteúdo Unicode ✓" {
		t.Fatal(string(b))
	}
	if e = f.Remove("sub/b.txt"); e != nil {
		t.Fatal(e)
	}
	if e = f.Remove("sub"); e != nil {
		t.Fatal(e)
	}
	if _, e = os.Stat(filepath.Join(dir, "sub")); !os.IsNotExist(e) {
		t.Fatal(e)
	}
}
