package app

import (
	"crypto/ed25519"
	"crypto/rand"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
	"net"
	"path/filepath"
	"termiashell/internal/core"
	"testing"
	"time"
)

func TestTrustRequiresPendingExactFingerprint(t *testing.T) {
	a, e := New(t.TempDir(), &testUI{})
	if e != nil {
		t.Fatal(e)
	}
	defer a.Shutdown()
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	key, _ := ssh.NewPublicKey(pub)
	p := core.Profile{Host: "127.0.0.1", Port: 2222}
	a.pending[core.Address(p)] = pendingTrust{key: key, created: time.Now(), remote: &net.TCPAddr{IP: net.ParseIP(p.Host), Port: p.Port}}
	if a.TrustHost(p.Host, p.Port, "SHA256:wrong") == nil {
		t.Fatal("accepted unverified fingerprint")
	}
	fp := ssh.FingerprintSHA256(key)
	if e = a.TrustHost(p.Host, p.Port, fp); e != nil {
		t.Fatal(e)
	}
	cb, e := knownhosts.New(filepath.Join(a.data, "known_hosts"))
	if e != nil {
		t.Fatal(e)
	}
	if e = cb(core.Address(p), &net.TCPAddr{IP: net.ParseIP(p.Host), Port: p.Port}, key); e != nil {
		t.Fatal(e)
	}
	if a.TrustHost(p.Host, p.Port, fp) == nil {
		t.Fatal("replayed trust request")
	}
}
