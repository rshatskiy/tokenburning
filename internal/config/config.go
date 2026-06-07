package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Config — постоянные настройки демона/отправки (~/.tokenburning/config.json).
type Config struct {
	IntervalMinutes int     `json:"intervalMinutes"`
	Push            PushCfg `json:"push"`
}

type PushCfg struct {
	Enabled    bool     `json:"enabled"`
	Categories []string `json:"categories"`
	Endpoint   string   `json:"endpoint"`
}

func (c Config) Interval() time.Duration {
	if c.IntervalMinutes <= 0 {
		return 15 * time.Minute
	}
	return time.Duration(c.IntervalMinutes) * time.Minute
}

// Path возвращает путь к конфигу: ~/.tokenburning/config.json.
func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".tokenburning", "config.json"), nil
}

// Load читает конфиг; отсутствующий файл — это значения по умолчанию (не ошибка).
func Load() (Config, error) {
	var c Config
	p, err := Path()
	if err != nil {
		return c, err
	}
	b, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return c, nil
	}
	if err != nil {
		return c, err
	}
	return c, json.Unmarshal(b, &c)
}

// Save атомарно записывает конфиг.
func Save(c Config) error {
	p, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, b, 0o600)
}
