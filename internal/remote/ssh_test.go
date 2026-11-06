package remote

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/pem"
	"fmt"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"termiashell/internal/core"
	"testing"
	"time"
)

// Servidor efêmero em loopback. Não abre uma shell do host e não instala serviço.
func testServer(t *testing.T) (core.Profile, ssh.Signer, ed25519.PrivateKey) {
	t.Helper()
	pub, key, _ := ed25519.GenerateKey(rand.Reader)
	_, host, _ := ed25519.GenerateKey(rand.Reader)
	signer, _ := ssh.NewSignerFromKey(host)
	cfg := &ssh.ServerConfig{PasswordCallback: func(c ssh.ConnMetadata, p []byte) (*ssh.Permissions, error) {
		if c.User() == "tester" && string(p) == "test-only" {
			return nil, nil
		}
		return nil, fmt.Errorf("denied")
	}, PublicKeyCallback: func(c ssh.ConnMetadata, k ssh.PublicKey) (*ssh.Permissions, error) {
		want, _ := ssh.NewPublicKey(pub)
		if bytes.Equal(want.Marshal(), k.Marshal()) {
			return nil, nil
		}
		return nil, fmt.Errorf("denied")
	}}
	cfg.AddHostKey(signer)
	ln, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { ln.Close() })
	root := filepath.ToSlash(t.TempDir())
	if !strings.HasPrefix(root, "/") {
		root = "/" + root
	}
	go func() {
		for {
			c, e := ln.Accept()
			if e != nil {
				return
			}
			go func() {
				defer c.Close()
				_, chs, reqs, e := ssh.NewServerConn(c, cfg)
				if e != nil {
					return
				}
				go ssh.DiscardRequests(reqs)
				for ch := range chs {
					if ch.ChannelType() != "session" {
						ch.Reject(ssh.UnknownChannelType, "session only")
						continue
					}
					s, rs, e := ch.Accept()
					if e != nil {
						continue
					}
					go func() {
						defer s.Close()
						for r := range rs {
							switch r.Type {
							case "pty-req", "window-change":
								r.Reply(true, nil)
							case "shell":
								r.Reply(true, nil)
								s.Write([]byte("READY\r\n"))
								go func() { io.Copy(s, s); s.Close() }()
							case "subsystem":
								var v struct{ Name string }
								ssh.Unmarshal(r.Payload, &v)
								if v.Name != "sftp" {
									r.Reply(false, nil)
									continue
								}
								r.Reply(true, nil)
								srv, e := sftp.NewServer(s, sftp.WithServerWorkingDirectory(root))
								if e == nil {
									srv.Serve()
									srv.Close()
								}
								return
							default:
								r.Reply(false, nil)
							}
						}
					}()
				}
			}()
		}
	}()
	_, port, _ := net.SplitHostPort(ln.Addr().String())
	n, _ := strconv.Atoi(port)
	return core.Profile{Name: "test", Kind: "ssh", Host: "127.0.0.1", Port: n, User: "tester", Directory: root}, signer, key
}
func TestSSHAndSFTP(t *testing.T) {
	p, host, key := testServer(t)
	check := ssh.FixedHostKey(host.PublicKey())
	term, fs, _, warn, e := Open(p, "test-only", "", check)
	if e != nil {
		t.Fatal(e)
	}
	defer term.Close()
	defer fs.Close()
	if warn != "" {
		t.Fatal(warn)
	}
	if e = term.Resize(120, 40); e != nil {
		t.Fatal(e)
	}
	done := make(chan string, 1)
	go func() { b := make([]byte, 128); n, _ := term.Read(b); done <- string(b[:n]) }()
	select {
	case out := <-done:
		if !strings.Contains(out, "READY") {
			t.Fatal(out)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no shell")
	}
	echo := make(chan string, 1)
	go func() { b := make([]byte, 128); n, _ := term.Read(b); echo <- string(b[:n]) }()
	term.Write([]byte("hello ✓"))
	select {
	case out := <-echo:
		if out != "hello ✓" {
			t.Fatal(out)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("echo timeout")
	}
	file := pathJoin(p.Directory, "payload.bin")
	w, e := fs.Create(file)
	if e != nil {
		t.Fatal(e)
	}
	payload := bytes.Repeat([]byte("Olá SSH ✓\n"), 10000)
	w.Write(payload)
	w.Close()
	r, e := fs.Open(file)
	if e != nil {
		t.Fatal(e)
	}
	received, e := io.ReadAll(r)
	r.Close()
	if e != nil || sha256.Sum256(payload) != sha256.Sum256(received) {
		t.Fatal("transfer hash mismatch", e)
	}
	if _, e = fs.Create(file); e == nil {
		t.Fatal("overwrote remote file")
	}
	renamed := file + ".renamed"
	if e = fs.Rename(file, renamed); e != nil {
		t.Fatal(e)
	}
	ls, e := fs.List(p.Directory)
	if e != nil || len(ls) != 1 {
		t.Fatal(ls, e)
	}
	if e = fs.Remove(renamed); e != nil {
		t.Fatal(e)
	}
	block, e := ssh.MarshalPrivateKeyWithPassphrase(key, "ephemeral", []byte("phrase"))
	if e != nil {
		t.Fatal(e)
	}
	keyPath := filepath.Join(t.TempDir(), "id_ed25519")
	os.WriteFile(keyPath, pem.EncodeToMemory(block), 0600)
	p.KeyPath = keyPath
	t2, f2, _, _, e := Open(p, "", "phrase", check)
	if e != nil {
		t.Fatal("key auth", e)
	}
	f2.Close()
	t2.Close()
	if _, _, _, _, e = Open(p, "", "wrong", check); e == nil {
		t.Fatal("wrong passphrase accepted")
	}
	p.KeyPath = ""
	if _, _, _, _, e = Open(p, "wrong", "", check); e == nil {
		t.Fatal("wrong password accepted")
	}
	_, other, _ := ed25519.GenerateKey(rand.Reader)
	wrong, _ := ssh.NewSignerFromKey(other)
	if _, _, _, _, e = Open(p, "test-only", "", ssh.FixedHostKey(wrong.PublicKey())); e == nil {
		t.Fatal("wrong host key accepted")
	}
}
func pathJoin(a, b string) string { return strings.TrimRight(a, "/") + "/" + b }
