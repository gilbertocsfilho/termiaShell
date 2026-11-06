package remote

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"golang.org/x/crypto/ssh"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"termiashell/internal/core"
	"testing"
	"time"
)

func TestContainerSSH(t *testing.T) {
	port := os.Getenv("TERMIASHELL_SSH_TEST_PORT")
	if port == "" {
		t.Skip("Container SSH não configurado")
	}
	n, e := strconv.Atoi(port)
	if e != nil {
		t.Fatal(e)
	}
	fingerprint := os.Getenv("TERMIASHELL_SSH_TEST_FINGERPRINT")
	if fingerprint == "" {
		t.Fatal("Fingerprint esperada obrigatória")
	}
	p := core.Profile{Name: "Container Linux", Kind: "ssh", Host: "127.0.0.1", Port: n, User: "tester"}
	term, fs, dir, warn, e := Open(p, "termiashell-test-only", "", func(_ string, _ net.Addr, key ssh.PublicKey) error {
		if ssh.FingerprintSHA256(key) != fingerprint {
			return fmt.Errorf("host fingerprint mismatch")
		}
		return nil
	})
	if e != nil {
		t.Fatal(e)
	}
	defer term.Close()
	defer fs.Close()
	if warn != "" {
		t.Fatal(warn)
	}
	if e = term.Resize(115, 37); e != nil {
		t.Fatal(e)
	}
	done := make(chan string, 1)
	go func() {
		var output strings.Builder
		b := make([]byte, 8192)
		for {
			n, e := term.Read(b)
			output.Write(b[:n])
			if strings.Contains(output.String(), "CONTAINER_OK_37 115") {
				done <- output.String()
				return
			}
			if e != nil {
				done <- output.String()
				return
			}
		}
	}()
	term.Write([]byte("printf 'CONTAINER_%s_' OK; stty size\r"))
	select {
	case out := <-done:
		if !strings.Contains(out, "CONTAINER_OK_37 115") {
			t.Fatal(out)
		}
	case <-time.After(8 * time.Second):
		t.Fatal("shell/resize timeout")
	}
	go io.Copy(io.Discard, term)
	file := dir + "/termiashell-validation.bin"
	payload := bytes.Repeat([]byte("Linux SFTP ✓\n"), 100000)
	w, e := fs.Create(file)
	if e != nil {
		t.Fatal(e)
	}
	defer fs.Remove(file)
	if _, e = w.Write(payload); e != nil {
		t.Fatal(e)
	}
	w.Close()
	r, e := fs.Open(file)
	if e != nil {
		t.Fatal(e)
	}
	received, e := io.ReadAll(r)
	r.Close()
	if e != nil || sha256.Sum256(payload) != sha256.Sum256(received) {
		t.Fatal("SFTP hash mismatch", e)
	}
	t.Log("PTY real Linux, resize e SFTP SHA-256 confirmados", len(payload), "bytes")
}
