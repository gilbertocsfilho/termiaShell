package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func csvTestApp(t *testing.T, dataDir, csvPath string) *App {
	t.Helper()
	a, err := New(dataDir, &testUI{open: csvPath})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(a.Shutdown)
	return a
}

func TestImportSSHProfilesCSVValidatesDeduplicatesAndIgnoresSecrets(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "servers.csv")
	contents := "name,host,port,username,key_path,password\n" +
		"App 1,example.com,2222,deploy,,secret-value\n" +
		"Duplicate,EXAMPLE.COM,2222,deploy,,another-secret\n" +
		"Bad host,invalid host,22,deploy,,secret-value\n"
	if err := os.WriteFile(file, []byte(contents), 0600); err != nil {
		t.Fatal(err)
	}
	a := csvTestApp(t, filepath.Join(root, "data"), file)
	result, err := a.ImportSSHProfilesCSV()
	if err != nil {
		t.Fatal(err)
	}
	if result.Imported != 1 || result.Skipped != 2 || len(result.Errors) != 1 {
		t.Fatalf("unexpected import result: %+v", result)
	}
	profiles, err := a.ListProfiles()
	if err != nil || len(profiles) != 1 {
		t.Fatalf("profiles=%+v err=%v", profiles, err)
	}
	if profiles[0].Port != 2222 || profiles[0].User != "deploy" || profiles[0].KeyPath != "" {
		t.Fatalf("unexpected profile: %+v", profiles[0])
	}
	saved, err := os.ReadFile(filepath.Join(root, "data", "profiles.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(saved), "secret-value") || strings.Contains(string(saved), "another-secret") {
		t.Fatal("CSV credentials were persisted")
	}
}

func TestImportSSHProfilesCSVBoundsAndValidatesInput(t *testing.T) {
	root := t.TempDir()
	large := filepath.Join(root, "large.csv")
	if err := os.WriteFile(large, make([]byte, maxProfileCSVBytes+1), 0600); err != nil {
		t.Fatal(err)
	}
	a := csvTestApp(t, filepath.Join(root, "data"), large)
	if _, err := a.ImportSSHProfilesCSV(); err == nil {
		t.Fatal("oversized CSV accepted")
	}

	missing := filepath.Join(root, "missing.csv")
	if err := os.WriteFile(missing, []byte("name,host\nserver,example.com\n"), 0600); err != nil {
		t.Fatal(err)
	}
	b := csvTestApp(t, filepath.Join(root, "other-data"), missing)
	if _, err := b.ImportSSHProfilesCSV(); err == nil {
		t.Fatal("CSV missing required username column accepted")
	}
	invalidUTF8 := filepath.Join(root, "invalid-utf8.csv")
	if err := os.WriteFile(invalidUTF8, []byte{'n', 'a', 'm', 'e', ',', 'h', 'o', 's', 't', ',', 'u', 's', 'e', 'r', '\n', 0xff}, 0600); err != nil {
		t.Fatal(err)
	}
	c := csvTestApp(t, filepath.Join(root, "utf8-data"), invalidUTF8)
	if _, err := c.ImportSSHProfilesCSV(); err == nil {
		t.Fatal("CSV with invalid UTF-8 accepted")
	}
}

func TestImportSSHProfilesCSVCancelDoesNotChangeStore(t *testing.T) {
	root := t.TempDir()
	a, err := New(filepath.Join(root, "data"), &testUI{})
	if err != nil {
		t.Fatal(err)
	}
	defer a.Shutdown()
	result, err := a.ImportSSHProfilesCSV()
	if err != nil || !result.Cancelled {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	profiles, err := a.ListProfiles()
	if err != nil || len(profiles) != 0 {
		t.Fatalf("profiles=%+v err=%v", profiles, err)
	}
}
