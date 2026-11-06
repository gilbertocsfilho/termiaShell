package app

import (
	"bytes"
	"crypto/sha256"
	"io"
	"os"
	"path/filepath"
	"sync"
	"termiashell/internal/localfiles"
	"testing"
	"time"
)

type testUI struct {
	open, save string
	events     chan map[string]interface{}
}

func (u *testUI) Emit(name string, args ...interface{}) {
	if name == "transfer:update" {
		u.events <- args[0].(map[string]interface{})
	}
}
func (u *testUI) OpenFile() (string, error)       { return u.open, nil }
func (u *testUI) OpenCSV() (string, error)        { return u.open, nil }
func (u *testUI) SaveFile(string) (string, error) { return u.save, nil }

type fakeTerm struct{ sync.Mutex }

func (*fakeTerm) Read([]byte) (int, error)    { return 0, io.EOF }
func (*fakeTerm) Write(b []byte) (int, error) { return len(b), nil }
func (*fakeTerm) Close() error                { return nil }
func (*fakeTerm) Resize(int, int) error       { return nil }
func waitTransfer(t *testing.T, u *testUI, tid string) map[string]interface{} {
	t.Helper()
	deadline := time.After(5 * time.Second)
	for {
		select {
		case ev := <-u.events:
			if ev["id"] == tid && ev["state"] != "running" {
				return ev
			}
		case <-deadline:
			t.Fatal("timeout transferência")
			return nil
		}
	}
}
func TestTransferRoundTripAndNoOverwrite(t *testing.T) {
	root := t.TempDir()
	remoteDir := filepath.Join(root, "remote")
	os.Mkdir(remoteDir, 0700)
	source := filepath.Join(root, "source.bin")
	data := bytes.Repeat([]byte("dados São Paulo ✓\n"), 100000)
	os.WriteFile(source, data, 0600)
	ui := &testUI{open: source, save: filepath.Join(root, "download.bin"), events: make(chan map[string]interface{}, 100)}
	a, e := New(filepath.Join(root, "data"), ui)
	if e != nil {
		t.Fatal(e)
	}
	defer a.Shutdown()
	files := &localfiles.Files{Resolve: func(p string) (string, error) { return filepath.Join(remoteDir, filepath.FromSlash(p)), nil }}
	sid := "test"
	a.sessions[sid] = &session{id: sid, term: &fakeTerm{}, files: files, closed: make(chan struct{}), ack: make(chan struct{}, 1)}
	tid, e := a.StartTransfer(sid, "/", "upload")
	if e != nil {
		t.Fatal(e)
	}
	ev := waitTransfer(t, ui, tid)
	if ev["state"] != "done" {
		t.Fatal(ev)
	}
	tid, e = a.StartTransfer(sid, "/source.bin", "download")
	if e != nil {
		t.Fatal(e)
	}
	ev = waitTransfer(t, ui, tid)
	if ev["state"] != "done" {
		t.Fatal(ev)
	}
	got, e := os.ReadFile(ui.save)
	if e != nil || sha256.Sum256(got) != sha256.Sum256(data) {
		t.Fatal("hash mismatch", e)
	}
	tid, _ = a.StartTransfer(sid, "/", "upload")
	ev = waitTransfer(t, ui, tid)
	if ev["state"] != "failed" {
		t.Fatal("overwrite accepted", ev)
	}
	entries, _ := os.ReadDir(remoteDir)
	if len(entries) != 1 {
		t.Fatal("temporary files leaked", entries)
	}
}
func TestPathValidation(t *testing.T) {
	for _, p := range []string{"relative", "/bad\x00name", "/bad\r\nname"} {
		if validPath(p) == nil {
			t.Fatal(p)
		}
	}
}
