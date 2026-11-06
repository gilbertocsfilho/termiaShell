package localfiles

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestSecurityRenameNoOverwriteUnderConcurrency(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, "destination")
	const count = 16
	start := make(chan struct{})
	wins := make(chan string, count)
	var wg sync.WaitGroup
	for i := 0; i < count; i++ {
		source := filepath.Join(root, fmt.Sprintf("source-%d", i))
		if e := os.WriteFile(source, []byte(source), 0600); e != nil {
			t.Fatal(e)
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if renameNoReplace(source, dest) == nil {
				wins <- source
			}
		}()
	}
	close(start)
	wg.Wait()
	close(wins)
	winners := []string{}
	for win := range wins {
		winners = append(winners, win)
	}
	if len(winners) != 1 {
		t.Fatalf("renomeação não foi exclusiva: %v", winners)
	}
	b, e := os.ReadFile(dest)
	if e != nil || string(b) != winners[0] {
		t.Fatal("destino sobrescrito", e)
	}
	entries, e := os.ReadDir(root)
	if e != nil || len(entries) != count {
		t.Fatal("fontes perdidas", e, len(entries))
	}
}
func TestSecuritySymlinkFinalAndAncestor(t *testing.T) {
	root := t.TempDir()
	real := filepath.Join(root, "real")
	os.Mkdir(real, 0700)
	os.WriteFile(filepath.Join(real, "secret"), []byte("do-not-follow"), 0600)
	if e := os.Symlink(real, filepath.Join(root, "link")); e != nil {
		t.Skip("Symlink não disponível neste ambiente:", e)
	}
	f := &Files{Resolve: func(p string) (string, error) { return filepath.Join(root, p), nil }}
	for _, p := range []string{"link", "link/secret"} {
		if _, e := f.Open(p); e == nil {
			t.Fatal("symlink foi seguido", p)
		}
	}
	if _, e := f.Create("link/new-file"); e == nil {
		t.Fatal("escrita através de symlink")
	}
}
