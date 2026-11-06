package app

import (
	"io"
	"os"
	"path/filepath"
	"sync"
	"termiashell/internal/core"
	"termiashell/internal/localfiles"
	"testing"
	"time"
)

type stalledFiles struct {
	core.Files
	started chan struct{}
}

func (f *stalledFiles) Create(p string) (io.WriteCloser, error) {
	w, e := f.Files.Create(p)
	if e != nil {
		return nil, e
	}
	return &stalledWriter{WriteCloser: w, started: f.started, closed: make(chan struct{})}, nil
}

type stalledWriter struct {
	io.WriteCloser
	started, closed      chan struct{}
	startOnce, closeOnce sync.Once
}

func (w *stalledWriter) Write([]byte) (int, error) {
	w.startOnce.Do(func() { close(w.started) })
	<-w.closed
	return 0, os.ErrClosed
}
func (w *stalledWriter) Close() error {
	var e error
	w.closeOnce.Do(func() { close(w.closed); e = w.WriteCloser.Close() })
	return e
}
func TestSecurityCancelRemovesOnlyOwnTemporary(t *testing.T) {
	root := t.TempDir()
	remote := filepath.Join(root, "remote")
	os.Mkdir(remote, 0700)
	source := filepath.Join(root, "payload")
	os.WriteFile(source, []byte("test payload"), 0600)
	keep := filepath.Join(remote, "existing")
	os.WriteFile(keep, []byte("keep me"), 0600)
	ui := &testUI{open: source, events: make(chan map[string]interface{}, 100)}
	a, e := New(filepath.Join(root, "data"), ui)
	if e != nil {
		t.Fatal(e)
	}
	defer a.Shutdown()
	fs := &stalledFiles{Files: &localfiles.Files{Resolve: func(p string) (string, error) { return filepath.Join(remote, filepath.FromSlash(p)), nil }}, started: make(chan struct{})}
	a.sessions["cancel"] = &session{term: &fakeTerm{}, files: fs, closed: make(chan struct{})}
	tid, e := a.StartTransfer("cancel", "/", "upload")
	if e != nil {
		t.Fatal(e)
	}
	select {
	case <-fs.started:
	case <-time.After(3 * time.Second):
		t.Fatal("escrita não iniciou")
	}
	a.CancelTransfer(tid)
	ev := waitTransfer(t, ui, tid)
	if ev["state"] != "failed" {
		t.Fatal("cancelamento ignorado", ev)
	}
	ls, e := os.ReadDir(remote)
	if e != nil || len(ls) != 1 || ls[0].Name() != "existing" {
		t.Fatal("temporário residual ou arquivo indevido removido", e, ls)
	}
	b, _ := os.ReadFile(keep)
	if string(b) != "keep me" {
		t.Fatal("arquivo original alterado")
	}
}
