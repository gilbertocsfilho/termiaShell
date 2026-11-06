package frontend_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func TestSecurityContentPolicyAndLocalScripts(t *testing.T) {
	root := "../.."
	b, e := os.ReadFile(filepath.Join(root, "frontend/index.html"))
	if e != nil {
		t.Fatal(e)
	}
	doc, e := html.Parse(bytes.NewReader(b))
	if e != nil {
		t.Fatal(e)
	}
	policyFound := false
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		attrs := map[string]string{}
		for _, a := range n.Attr {
			attrs[a.Key] = a.Val
		}
		if n.Type == html.ElementNode && n.Data == "meta" && strings.EqualFold(attrs["http-equiv"], "Content-Security-Policy") {
			policyFound = true
			p := attrs["content"]
			for _, required := range []string{"default-src 'self'", "script-src 'self'", "connect-src 'self'", "object-src 'none'", "base-uri 'none'", "form-action 'none'"} {
				if !strings.Contains(p, required) {
					t.Error("diretiva CSP ausente", required)
				}
			}
			if strings.Contains(p, "unsafe-eval") {
				t.Error("CSP permite eval")
			}
		}
		if n.Type == html.ElementNode && n.Data == "script" {
			src := attrs["src"]
			if src == "" || strings.ContainsAny(src, ":\\") || strings.HasPrefix(src, "//") || strings.Contains(src, "..") {
				t.Error("script inline/remoto", src)
			}
			if _, e := os.Stat(filepath.Join(root, "frontend", src)); e != nil {
				t.Error("script ausente", src, e)
			}
		}
		if n.Type == html.ElementNode {
			for name := range attrs {
				if strings.HasPrefix(strings.ToLower(name), "on") {
					t.Error("handler inline", name)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	if !policyFound {
		t.Fatal("CSP ausente")
	}
}
func TestSecurityVendoredAssetIntegrity(t *testing.T) {
	b, e := os.ReadFile("../../third_party/assets-sha256.json")
	if e != nil {
		t.Fatal(e)
	}
	b = bytes.TrimPrefix(b, []byte{239, 187, 191})
	var entries []struct{ File, SHA256 string }
	if e = json.Unmarshal(b, &entries); e != nil {
		t.Fatal(e)
	}
	if len(entries) != 3 {
		t.Fatal("manifesto de assets inesperado")
	}
	for _, entry := range entries {
		if filepath.Base(entry.File) != entry.File {
			t.Fatal("caminho adulterado no manifesto")
		}
		b, e := os.ReadFile(filepath.Join("../../frontend/vendor", entry.File))
		if e != nil {
			t.Fatal(e)
		}
		sum := sha256.Sum256(b)
		if !strings.EqualFold(hex.EncodeToString(sum[:]), entry.SHA256) {
			t.Error("hash de asset divergente", entry.File)
		}
	}
}
