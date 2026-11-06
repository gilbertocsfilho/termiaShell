package core

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"
	"unicode"
)

type Profile struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Host      string `json:"host"`
	Port      int    `json:"port"`
	User      string `json:"user"`
	KeyPath   string `json:"keyPath"`
	Target    string `json:"target"`
	Directory string `json:"directory"`
}

func (p Profile) Validate() error {
	if strings.TrimSpace(p.Name) == "" || len(p.Name) > 120 {
		return errors.New("Informe um nome de até 120 caracteres")
	}
	for _, v := range []string{p.Name, p.Host, p.User, p.Target, p.Directory, p.KeyPath} {
		if len(v) > 32768 {
			return errors.New("Campo contém caracteres inválidos")
		}
		for _, r := range v {
			if unicode.IsControl(r) {
				return errors.New("Campo contém caracteres de controle")
			}
		}
	}
	if p.Directory != "" {
		if e := ValidateFilePath(p.Directory); e != nil {
			return e
		}
	}
	switch p.Kind {
	case "wsl", "local":
		return nil
	case "ssh":
		if strings.TrimSpace(p.User) == "" || len(p.User) > 256 {
			return errors.New("Usuário SSH inválido")
		}
		if p.Host == "" || len(p.Host) > 253 || strings.HasPrefix(p.Host, "-") || strings.ContainsAny(p.Host, " /\\\t") {
			return errors.New("Host inválido")
		}
		if net.ParseIP(strings.Trim(p.Host, "[]")) == nil && strings.Contains(p.Host, ":") {
			return errors.New("Informe a porta no campo separado")
		}
		if p.Port < 1 || p.Port > 65535 {
			return errors.New("Porta inválida")
		}
		return nil
	default:
		return errors.New("Tipo de sessão não suportado")
	}
}
func Address(p Profile) string {
	return net.JoinHostPort(strings.Trim(p.Host, "[]"), fmt.Sprint(p.Port))
}

type Entry struct {
	Name      string    `json:"name"`
	Directory bool      `json:"directory"`
	Link      bool      `json:"link"`
	Size      int64     `json:"size"`
	Modified  time.Time `json:"modified"`
	Mode      string    `json:"mode"`
}
type Target struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Kind string `json:"kind"`
}
type Capabilities struct {
	OS        string `json:"os"`
	LocalKind string `json:"localKind"`
}
type Terminal interface {
	io.ReadWriteCloser
	Resize(int, int) error
}
type Files interface {
	List(string) ([]Entry, error)
	Open(string) (io.ReadCloser, error)
	Create(string) (io.WriteCloser, error)
	Mkdir(string) error
	Rename(string, string) error
	Chmod(string, os.FileMode) error
	Remove(string) error
	Exists(string) (bool, error)
	Close() error
}
