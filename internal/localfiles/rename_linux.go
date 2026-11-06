package localfiles

import (
	"errors"
	"golang.org/x/sys/unix"
	"os"
)

func renameNoReplace(from, to string) error {
	// Sem fallback para Rename comum: ele poderia sobrescrever um destino criado
	// por outro processo entre a verificação e a operação.
	e := unix.Renameat2(unix.AT_FDCWD, from, unix.AT_FDCWD, to, unix.RENAME_NOREPLACE)
	if !errors.Is(e, unix.ENOSYS) && !errors.Is(e, unix.EINVAL) && !errors.Is(e, unix.EOPNOTSUPP) {
		return e
	}
	// DrvFS pode não implementar RENAME_NOREPLACE. Hard link cria o destino
	// exclusivamente; nunca substituir essa operação por Rename comum.
	st, err := os.Lstat(from)
	if err != nil {
		return err
	}
	if !st.Mode().IsRegular() {
		return errors.New("Filesystem não suporta renomeação de diretório sem sobrescrita")
	}
	if err = os.Link(from, to); err != nil {
		return err
	}
	// Se remover a origem falhar, preservar ambos os nomes em vez de arriscar
	// apagar um destino que outro processo possa ter alterado.
	return os.Remove(from)
}
