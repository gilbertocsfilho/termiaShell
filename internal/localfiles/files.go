package localfiles

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"termiashell/internal/core"
)

type Files struct{ Resolve func(string) (string, error) }

func (f *Files) path(p string) (string, error) {
	v, e := f.Resolve(p)
	if e != nil {
		return "", e
	}
	cur := v
	for {
		st, e := os.Lstat(cur)
		if e == nil && st.Mode()&os.ModeSymlink != 0 {
			return "", errors.New("Links simbólicos não são seguidos")
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}
	return v, nil
}
func (f *Files) List(p string) ([]core.Entry, error) {
	v, e := f.path(p)
	if e != nil {
		return nil, e
	}
	ds, e := os.ReadDir(v)
	if e != nil {
		return nil, e
	}
	out := []core.Entry{}
	for _, d := range ds {
		st, e := d.Info()
		if e != nil {
			continue
		}
		out = append(out, core.Entry{Name: d.Name(), Directory: d.IsDir(), Link: st.Mode()&os.ModeSymlink != 0, Size: st.Size(), Modified: st.ModTime(), Mode: fmt.Sprintf("%04o", st.Mode().Perm())})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Directory != out[j].Directory {
			return out[i].Directory
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}
func (f *Files) Open(p string) (io.ReadCloser, error) {
	v, e := f.path(p)
	if e != nil {
		return nil, e
	}
	st, e := os.Stat(v)
	if e != nil {
		return nil, e
	}
	if !st.Mode().IsRegular() {
		return nil, errors.New("Selecione um arquivo regular")
	}
	return os.Open(v)
}
func (f *Files) Create(p string) (io.WriteCloser, error) {
	v, e := f.path(p)
	if e != nil {
		return nil, e
	}
	return os.OpenFile(v, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
}
func (f *Files) Mkdir(p string) error {
	v, e := f.path(p)
	if e != nil {
		return e
	}
	return os.Mkdir(v, 0755)
}
func (f *Files) Rename(a, b string) error {
	x, e := f.path(a)
	if e != nil {
		return e
	}
	y, e := f.path(b)
	if e != nil {
		return e
	}
	if _, e = os.Lstat(y); !errors.Is(e, os.ErrNotExist) {
		return errors.New("Destino já existe ou não pode ser verificado")
	}
	return renameNoReplace(x, y)
}
func (f *Files) Chmod(p string, mode os.FileMode) error {
	v, e := f.path(p)
	if e != nil {
		return e
	}
	return os.Chmod(v, mode)
}
func (f *Files) Remove(p string) error {
	v, e := f.path(p)
	if e != nil {
		return e
	}
	return os.Remove(v)
}
func (f *Files) Exists(p string) (bool, error) {
	v, e := f.path(p)
	if e != nil {
		return false, e
	}
	_, e = os.Lstat(v)
	if errors.Is(e, os.ErrNotExist) {
		return false, nil
	}
	return e == nil, e
}
func (f *Files) Close() error { return nil }
