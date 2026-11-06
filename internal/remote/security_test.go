package remote

import "testing"

func TestSecurityDotSegmentsRejectedBeforeSFTP(t *testing.T) {
	f := &Files{}
	for _, p := range []string{"/link/../victim", "/link/./victim", "/bad\x00file"} {
		if _, e := f.Open(p); e == nil {
			t.Fatal("caminho ambíguo aceito")
		}
		if _, e := f.Create(p); e == nil {
			t.Fatal("escrita ambígua aceita")
		}
	}
}
