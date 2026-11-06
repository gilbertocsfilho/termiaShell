package app

import (
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
	"net"
	"os"
	"path/filepath"
	"termiashell/internal/core"
	"testing"
	"time"
)

func newHostKey(t *testing.T) ssh.PublicKey {
	t.Helper()
	pub, _, e := ed25519.GenerateKey(rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	key, e := ssh.NewPublicKey(pub)
	if e != nil {
		t.Fatal(e)
	}
	return key
}
func TestSecurityExpiredTrustAndPendingLimit(t *testing.T) {
	a, e := New(t.TempDir(), &testUI{})
	if e != nil {
		t.Fatal(e)
	}
	defer a.Shutdown()
	key := newHostKey(t)
	addr := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 22}
	p := core.Profile{Name: "test", Kind: "ssh", Host: "127.0.0.1", Port: 22, User: "tester"}
	a.pending[core.Address(p)] = pendingTrust{key: key, created: time.Now().Add(-trustLifetime - time.Second), remote: addr}
	if a.TrustHost(p.Host, p.Port, ssh.FingerprintSHA256(key)) == nil {
		t.Fatal("confirmação expirada aceita")
	}
	if len(a.pending) != 0 {
		t.Fatal("confirmação expirada retida")
	}
	for i := 0; i < maxPendingTrust+10; i++ {
		p.Host = fmt.Sprintf("host-%d.test", i)
		a.hostKeyCallback(p)(core.Address(p), addr, key)
	}
	if len(a.pending) != maxPendingTrust {
		t.Fatal("fila sem limite", len(a.pending))
	}
}
func TestSecurityHostChangedWhileConfirmationOpen(t *testing.T) {
	a, e := New(t.TempDir(), &testUI{})
	if e != nil {
		t.Fatal(e)
	}
	defer a.Shutdown()
	presented, other := newHostKey(t), newHostKey(t)
	p := core.Profile{Name: "test", Kind: "ssh", Host: "127.0.0.1", Port: 2222, User: "tester"}
	addr := &net.TCPAddr{IP: net.ParseIP(p.Host), Port: p.Port}
	if a.hostKeyCallback(p)(core.Address(p), addr, presented) == nil {
		t.Fatal("host desconhecido aceito sem confirmação")
	}
	line := knownhosts.Line([]string{knownhosts.Normalize(core.Address(p))}, other) + "\n"
	file := filepath.Join(a.data, "known_hosts")
	os.WriteFile(file, []byte(line), 0600)
	if a.TrustHost(p.Host, p.Port, ssh.FingerprintSHA256(presented)) == nil {
		t.Fatal("chave alterada aceita")
	}
	got, _ := os.ReadFile(file)
	if string(got) != line {
		t.Fatal("confiança alterada indevidamente")
	}
	if a.hostKeyCallback(p)(core.Address(p), addr, presented) == nil {
		t.Fatal("MITM aceito")
	}
}
