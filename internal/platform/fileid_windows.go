//go:build windows

package platform

import (
	"golang.org/x/sys/windows"
)

// Stat возвращает идентичность файла из volume serial + file index.
func Stat(path string) (FileID, error) {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return FileID{}, err
	}
	h, err := windows.CreateFile(p, 0, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil, windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		return FileID{}, err
	}
	defer windows.CloseHandle(h)
	var info windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(h, &info); err != nil {
		return FileID{}, err
	}
	index := uint64(info.FileIndexHigh)<<32 | uint64(info.FileIndexLow)
	return FileID{A: uint64(info.VolumeSerialNumber), B: index}, nil
}
