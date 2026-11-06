package core

import (
	"fmt"
	"strings"
	"testing"
)

func TestSecurityRejectAmbiguousPaths(t *testing.T) {
	for i, p := range []string{"relative", "", "/link/../victim", "/safe/./file", "/../secret", `/safe\..\secret`, "/bad\x00name", "/bad\x1b[2J", "/bad\nname", strings.Repeat("a", 32769), "/bad\xff"} {
		t.Run(fmt.Sprintf("input-%02d", i), func(t *testing.T) {
			if ValidateFilePath(p) == nil {
				t.Fatal("aceitou caminho perigoso")
			}
		})
	}
	for _, p := range []string{"/", "/home/user/file.txt", "/tmp/São Paulo/arquivo.txt"} {
		if e := ValidateFilePath(p); e != nil {
			t.Fatalf("caminho legítimo rejeitado: %v", e)
		}
	}
}
func TestSecurityRejectRemoteEntryTraversal(t *testing.T) {
	for _, name := range []string{"", ".", "..", "../secret", "a/b", `a\b`, "bad\x00name", "bad\nname", "bad\x1bname"} {
		if ValidateEntryName(name) == nil {
			t.Fatalf("nome inválido aceito: %q", name)
		}
	}
	if e := ValidateEntryName("São Paulo.txt"); e != nil {
		t.Fatal(e)
	}
}
func TestSecurityProfileInput(t *testing.T) {
	base := Profile{Name: "SSH", Kind: "ssh", User: "tester", Host: "example.test", Port: 22}
	for _, mutate := range []func(*Profile){func(p *Profile) { p.User = "" }, func(p *Profile) { p.Host = "-oProxyCommand=bad" }, func(p *Profile) { p.Host = "host\r\nHost evil" }, func(p *Profile) { p.Host = "host\x1b[2J" }, func(p *Profile) { p.Directory = "/link/../victim" }, func(p *Profile) { p.Kind = "vnc" }, func(p *Profile) { p.Port = -1 }, func(p *Profile) { p.KeyPath = "key\x00file" }} {
		p := base
		mutate(&p)
		if p.Validate() == nil {
			t.Fatalf("perfil inválido aceito: %#v", p)
		}
	}
}
func FuzzFilePathValidation(f *testing.F) {
	for _, p := range []string{"/safe/file", "/link/../victim", "/\x00", "/São Paulo", "/bad\xff"} {
		f.Add(p)
	}
	f.Fuzz(func(t *testing.T, p string) {
		if ValidateFilePath(p) == nil {
			for _, part := range strings.Split(p, "/") {
				if part == "." || part == ".." {
					t.Fatal("caminho ambíguo aceito")
				}
			}
			if len(p) > 32768 {
				t.Fatal("limite excedido")
			}
		}
	})
}
