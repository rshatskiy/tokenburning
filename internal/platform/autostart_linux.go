//go:build linux

package platform

import (
	"os"
	"os/exec"
	"path/filepath"
)

func unitPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "systemd", "user", "tokenburning.service"), nil
}

// unitContent строит systemd user-unit (чистая функция — тестируемо).
func unitContent(exe string) string {
	return `[Unit]
Description=tokenburning background collector

[Service]
ExecStart=` + exe + ` daemon
Restart=on-failure

[Install]
WantedBy=default.target
`
}

func EnableAutostart(exe string) error {
	p, err := unitPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(p, []byte(unitContent(exe)), 0o644); err != nil {
		return err
	}
	_ = exec.Command("systemctl", "--user", "daemon-reload").Run()
	return exec.Command("systemctl", "--user", "enable", "--now", "tokenburning.service").Run()
}

func DisableAutostart() error {
	_ = exec.Command("systemctl", "--user", "disable", "--now", "tokenburning.service").Run()
	p, err := unitPath()
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func AutostartInstalled() (bool, string) {
	p, err := unitPath()
	if err != nil {
		return false, ""
	}
	if _, err := os.Stat(p); err == nil {
		return true, p
	}
	return false, p
}
