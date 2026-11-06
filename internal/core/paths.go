package core

import (
	"errors"
	"path"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Caminhos lógicos POSIX. Rejeitar segmentos ambíguos antes de Lstat/normalização
// evita validar um caminho e operar outro quando há links no trajeto original.
func ValidateFilePath(p string) error {
	if len(p) > 32768 || !utf8.ValidString(p) || !path.IsAbs(p) || strings.ContainsRune(p, '\\') {
		return errors.New("Caminho absoluto inválido")
	}
	for _, r := range p {
		if unicode.IsControl(r) {
			return errors.New("Caminho contém caracteres de controle")
		}
	}
	for _, part := range strings.Split(p, "/") {
		if part == "." || part == ".." {
			return errors.New("Use o caminho absoluto sem segmentos . ou ..")
		}
	}
	return nil
}

func ValidateEntryName(name string) error {
	if name == "" || name == "." || name == ".." || !utf8.ValidString(name) || strings.ContainsAny(name, "/\\") {
		return errors.New("Servidor retornou nome de arquivo inválido")
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return errors.New("Nome de arquivo contém caracteres de controle")
		}
	}
	return nil
}
