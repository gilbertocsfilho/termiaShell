package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"termiashell/internal/core"
)

type Store struct {
	Dir string
	mu  sync.Mutex
}
type document struct {
	Version  int            `json:"schemaVersion"`
	Profiles []core.Profile `json:"profiles"`
}

func (s *Store) Load() ([]core.Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.load()
}
func (s *Store) load() ([]core.Profile, error) {
	b, e := os.ReadFile(filepath.Join(s.Dir, "profiles.json"))
	if errors.Is(e, os.ErrNotExist) {
		return []core.Profile{}, nil
	}
	if e != nil {
		return nil, e
	}
	var d document
	if e = json.Unmarshal(b, &d); e != nil {
		return nil, e
	}
	if d.Version != 1 {
		return nil, errors.New("Versão de perfis incompatível")
	}
	if d.Profiles == nil {
		d.Profiles = []core.Profile{}
	}
	return d.Profiles, nil
}
func (s *Store) Save(p core.Profile) error {
	if e := p.Validate(); e != nil {
		return e
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	ps, e := s.load()
	if e != nil {
		return e
	}
	found := false
	for i := range ps {
		if ps[i].ID == p.ID {
			ps[i] = p
			found = true
		}
	}
	if !found {
		ps = append(ps, p)
	}
	return s.write(ps)
}
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	ps, e := s.load()
	if e != nil {
		return e
	}
	out := []core.Profile{}
	for _, p := range ps {
		if p.ID != id {
			out = append(out, p)
		}
	}
	return s.write(out)
}
func (s *Store) write(ps []core.Profile) error {
	b, e := json.MarshalIndent(document{1, ps}, "", "  ")
	if e != nil {
		return e
	}
	f, e := os.CreateTemp(s.Dir, "profiles-*.tmp")
	if e != nil {
		return e
	}
	name := f.Name()
	defer os.Remove(name)
	if e = f.Chmod(0600); e != nil {
		f.Close()
		return e
	}
	if _, e = f.Write(b); e != nil {
		f.Close()
		return e
	}
	if e = f.Sync(); e != nil {
		f.Close()
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	return replace(name, filepath.Join(s.Dir, "profiles.json"))
}
