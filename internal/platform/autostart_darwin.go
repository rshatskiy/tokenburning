//go:build darwin

package platform

import (
	"os"
	"os/exec"
	"path/filepath"
)

func plistPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "LaunchAgents", autostartLabel+".plist"), nil
}

// plistContent строит содержимое LaunchAgent plist (чистая функция — тестируемо).
func plistContent(exe, logPath string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key><string>` + autostartLabel + `</string>
  <key>ProgramArguments</key>
  <array>
    <string>` + exe + `</string>
    <string>daemon</string>
  </array>
  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key><true/>
  <key>StandardOutPath</key><string>` + logPath + `</string>
  <key>StandardErrorPath</key><string>` + logPath + `</string>
</dict>
</plist>
`
}

func EnableAutostart(exe string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	p, err := plistPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	logPath := filepath.Join(home, ".tokenburning", "daemon.log")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(p, []byte(plistContent(exe, logPath)), 0o644); err != nil {
		return err
	}
	// перезагрузить агент (idempotent): unload (игнор ошибки) затем load -w
	_ = exec.Command("launchctl", "unload", p).Run()
	return exec.Command("launchctl", "load", "-w", p).Run()
}

func DisableAutostart() error {
	p, err := plistPath()
	if err != nil {
		return err
	}
	_ = exec.Command("launchctl", "unload", "-w", p).Run()
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func AutostartInstalled() (bool, string) {
	p, err := plistPath()
	if err != nil {
		return false, ""
	}
	if _, err := os.Stat(p); err == nil {
		return true, p
	}
	return false, p
}
