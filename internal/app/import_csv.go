package app

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"termiashell/internal/core"
)

const maxProfileCSVBytes = 2 << 20
const maxImportedProfiles = 5000

type CSVImportResult struct {
	Imported  int      `json:"imported"`
	Skipped   int      `json:"skipped"`
	Errors    []string `json:"errors"`
	Cancelled bool     `json:"cancelled"`
}

func profileIdentity(p core.Profile) string {
	return strings.ToLower(strings.TrimSpace(p.Host)) + "\x00" + strconv.Itoa(p.Port) + "\x00" + p.User
}

// ImportSSHProfilesCSV opens a user-selected CSV. Passwords and other credential columns are ignored.
func (a *App) ImportSSHProfilesCSV() (CSVImportResult, error) {
	a.csvMu.Lock()
	defer a.csvMu.Unlock()
	file, err := a.ui.OpenCSV()
	if err != nil {
		return CSVImportResult{}, err
	}
	if file == "" {
		return CSVImportResult{Cancelled: true}, nil
	}
	return a.importSSHProfilesCSV(file)
}

func (a *App) importSSHProfilesCSV(file string) (CSVImportResult, error) {
	var result CSVImportResult
	if !strings.EqualFold(filepath.Ext(file), ".csv") {
		return result, errors.New("Selecione um arquivo .csv")
	}
	f, err := os.Open(file)
	if err != nil {
		return result, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return result, err
	}
	if !info.Mode().IsRegular() || info.Size() > maxProfileCSVBytes {
		return result, errors.New("O CSV deve ser um arquivo regular de até 2 MB")
	}
	data, err := io.ReadAll(io.LimitReader(f, maxProfileCSVBytes+1))
	if err != nil {
		return result, err
	}
	if len(data) > maxProfileCSVBytes {
		return result, errors.New("O CSV excede 2 MB")
	}
	if !utf8.Valid(data) {
		return result, errors.New("O CSV precisa estar codificado em UTF-8")
	}
	r := csv.NewReader(bytes.NewReader(data))
	r.FieldsPerRecord = -1
	r.TrimLeadingSpace = true
	rows, err := r.ReadAll()
	if err != nil {
		return result, fmt.Errorf("CSV inválido: %w", err)
	}
	if len(rows) < 2 {
		return result, errors.New("O CSV precisa de cabeçalho e ao menos uma linha")
	}
	if len(rows)-1 > maxImportedProfiles {
		return result, fmt.Errorf("O limite por importação é %d conexões", maxImportedProfiles)
	}
	indexes := make(map[string]int)
	for i, h := range rows[0] {
		name := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(h, "\ufeff")))
		if _, exists := indexes[name]; !exists {
			indexes[name] = i
		}
	}
	aliases := map[string][]string{
		"name":      {"name", "label"},
		"host":      {"host", "hostname", "host/ip", "address", "ip"},
		"user":      {"user", "username", "login"},
		"port":      {"port"},
		"keypath":   {"keypath", "key_path", "privatekey", "private_key"},
		"directory": {"directory", "path", "initialdirectory"},
	}
	canonical := make(map[string]int)
	for field, options := range aliases {
		for _, option := range options {
			if i, ok := indexes[option]; ok {
				canonical[field] = i
				break
			}
		}
	}
	for _, required := range []string{"name", "host", "user"} {
		if _, ok := canonical[required]; !ok {
			return result, fmt.Errorf("Coluna obrigatória ausente: %s", required)
		}
	}
	get := func(row []string, key string) string {
		i, ok := canonical[key]
		if !ok || i >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[i])
	}
	existing, err := a.store.Load()
	if err != nil {
		return result, err
	}
	seen := make(map[string]struct{}, len(existing)+len(rows)-1)
	for _, p := range existing {
		if p.Kind == "ssh" {
			seen[profileIdentity(p)] = struct{}{}
		}
	}
	for i, row := range rows[1:] {
		line := i + 2
		if len(row) == 0 || strings.TrimSpace(strings.Join(row, "")) == "" {
			continue
		}
		port := 22
		if raw := get(row, "port"); raw != "" {
			port, err = strconv.Atoi(raw)
			if err != nil {
				result.addError(line, "porta inválida")
				continue
			}
		}
		p := core.Profile{ID: id(), Name: get(row, "name"), Kind: "ssh", Host: get(row, "host"), Port: port, User: get(row, "user"), KeyPath: get(row, "keypath"), Directory: get(row, "directory")}
		if err = p.Validate(); err != nil {
			result.addError(line, err.Error())
			continue
		}
		key := profileIdentity(p)
		if _, duplicate := seen[key]; duplicate {
			result.Skipped++
			continue
		}
		if err = a.store.Save(p); err != nil {
			result.addError(line, err.Error())
			continue
		}
		seen[key] = struct{}{}
		result.Imported++
	}
	return result, nil
}

func (r *CSVImportResult) addError(line int, message string) {
	r.Skipped++
	if len(r.Errors) < 20 {
		r.Errors = append(r.Errors, fmt.Sprintf("Linha %d: %s", line, message))
	}
}
