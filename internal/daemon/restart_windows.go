//go:build windows

package daemon

// На Windows нельзя exec-нуть поверх работающего процесса; новый бинарь уже записан на
// диск и применится при следующем запуске задачи (вход в систему). Просто завершаемся.
func restartSelf() error { return nil }
