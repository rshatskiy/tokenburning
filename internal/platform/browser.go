package platform

import (
	"os/exec"
	"runtime"
)

// browserCommand возвращает команду и аргументы для открытия URL в браузере на текущей ОС.
func browserCommand(url string) (string, []string) {
	switch runtime.GOOS {
	case "darwin":
		return "open", []string{url}
	case "windows":
		return "rundll32", []string{"url.dll,FileProtocolHandler", url}
	default:
		return "xdg-open", []string{url}
	}
}

// OpenBrowser пытается открыть URL в браузере по умолчанию. Ошибка не фатальна
// (пользователь может открыть URL вручную) — возвращается вызывающему для логирования.
func OpenBrowser(url string) error {
	name, args := browserCommand(url)
	return exec.Command(name, args...).Start()
}
