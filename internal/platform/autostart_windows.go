//go:build windows

package platform

import (
	"os/exec"
)

const taskName = "tokenburning"

// taskRunArg строит значение /TR (чистая функция — тестируемо).
func taskRunArg(exe string) string {
	return `"` + exe + `" daemon`
}

func EnableAutostart(exe string) error {
	return exec.Command("schtasks", "/Create", "/TN", taskName, "/TR", taskRunArg(exe), "/SC", "ONLOGON", "/F").Run()
}

func DisableAutostart() error {
	return exec.Command("schtasks", "/Delete", "/TN", taskName, "/F").Run()
}

func AutostartInstalled() (bool, string) {
	err := exec.Command("schtasks", "/Query", "/TN", taskName).Run()
	return err == nil, taskName
}
