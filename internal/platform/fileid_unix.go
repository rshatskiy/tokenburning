//go:build !windows

package platform

import (
	"os"
	"syscall"
)

// Stat возвращает идентичность файла из inode+dev.
func Stat(path string) (FileID, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return FileID{}, err
	}
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return FileID{}, nil // неизвестная платформа — нулевой ID (fallback на хеш позже)
	}
	return FileID{A: uint64(int64(st.Dev)), B: uint64(st.Ino)}, nil
}
