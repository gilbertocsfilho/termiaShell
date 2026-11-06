package localfiles

import "golang.org/x/sys/windows"

func renameNoReplace(from, to string) error {
	a, e := windows.UTF16PtrFromString(from)
	if e != nil {
		return e
	}
	b, e := windows.UTF16PtrFromString(to)
	if e != nil {
		return e
	}
	// Ausência de MOVEFILE_REPLACE_EXISTING garante a recusa no próprio sistema.
	return windows.MoveFileEx(a, b, windows.MOVEFILE_WRITE_THROUGH)
}
