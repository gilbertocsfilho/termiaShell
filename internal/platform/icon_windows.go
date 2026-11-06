//go:build windows

package platform

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

func SetAppIcon(path string) {
	title, err := windows.UTF16PtrFromString("termiaShell")
	if err != nil {
		return
	}
	hwnd, _, _ := windows.NewLazySystemDLL("user32.dll").NewProc("FindWindowW").Call(0, uintptr(unsafe.Pointer(title)))
	if hwnd == 0 {
		return
	}
	file, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return
	}
	icon, _, _ := windows.NewLazySystemDLL("user32.dll").NewProc("LoadImageW").Call(0, uintptr(unsafe.Pointer(file)), 1, 0, 0, 0x10|0x40)
	if icon == 0 {
		return
	}
	sendMessage := windows.NewLazySystemDLL("user32.dll").NewProc("SendMessageW")
	sendMessage.Call(hwnd, 0x80, 0, icon) // WM_SETICON / ICON_SMALL
	sendMessage.Call(hwnd, 0x80, 1, icon) // WM_SETICON / ICON_BIG
}
