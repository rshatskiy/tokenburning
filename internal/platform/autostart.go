package platform

// autostartLabel — стабильный идентификатор автозапуска демона.
const autostartLabel = "com.tokenburning.daemon"

// Autostart описывает регистрацию фонового демона в автозапуске ОС.
// Реализации — в build-tagged файлах autostart_<os>.go.
//   EnableAutostart(exe) — зарегистрировать `<exe> daemon` на автозапуск (без root).
//   DisableAutostart()   — снять регистрацию.
//   AutostartInstalled() — установлен ли (true) и путь к конфигу/описание.
