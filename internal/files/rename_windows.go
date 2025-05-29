//go:build windows

package files

import (
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modkernel32      = syscall.NewLazyDLL("kernel32.dll")
	procReplaceFileW = modkernel32.NewProc("ReplaceFileW")
)

func replaceFileW(replacedFileName, newFileName, backupFileName *uint16, dwReplaceFlags uint32, exclude, reserved unsafe.Pointer) error {
	r1, _, err := syscall.SyscallN(
		procReplaceFileW.Addr(),
		uintptr(unsafe.Pointer(replacedFileName)),
		uintptr(unsafe.Pointer(newFileName)),
		uintptr(unsafe.Pointer(backupFileName)),
		uintptr(dwReplaceFlags),
		uintptr(exclude),
		uintptr(reserved),
	)
	if r1 == 0 {
		return err
	}
	return nil
}

func replaceFile(oldPath, newPath string) error {
	replacedFileName, err := windows.UTF16PtrFromString(oldPath)
	if err != nil {
		return err
	}

	newFileName, err := windows.UTF16PtrFromString(newPath)
	if err != nil {
		return err
	}

	return replaceFileW(replacedFileName, newFileName, nil, 0, nil, nil)
}

func rename(oldPath, newPath string) error {
	err := replaceFile(oldPath, newPath)
	if err != nil && err != windows.ERROR_FILE_NOT_FOUND {
		return err
	}

	return os.Rename(oldPath, newPath)
}
