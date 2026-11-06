package core

import "testing"

func TestProfileValidation(t *testing.T) {
	for _, p := range []Profile{{Name: "ok", Kind: "ssh", User: "tester", Host: "host\r\npassword:s:bad", Port: 22}, {Name: "ok", Kind: "ssh", User: "tester", Host: "host", Port: 70000}, {Name: "ok", Kind: "ftp", Host: "host", Port: 21}} {
		if p.Validate() == nil {
			t.Fatalf("accepted invalid profile: %#v", p)
		}
	}
	p := Profile{Name: "IPv6", Kind: "ssh", User: "tester", Host: "::1", Port: 2222}
	if e := p.Validate(); e != nil {
		t.Fatal(e)
	}
	if Address(p) != "[::1]:2222" {
		t.Fatal(Address(p))
	}
}
