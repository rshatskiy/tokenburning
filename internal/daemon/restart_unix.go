//go:build !windows

package daemon

import (
	"os"
	"syscall"
)

// restartSelf заменяет образ текущего процесса свежим бинарём (тот же PID) — корректно
// под управлением launchd/systemd (сервис-менеджер продолжает следить за тем же процессом).
func restartSelf() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	return syscall.Exec(exe, os.Args, os.Environ())
}
