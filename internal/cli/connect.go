package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/rshatskiy/tokenburning/internal/aggregate"
	"github.com/rshatskiy/tokenburning/internal/config"
	"github.com/rshatskiy/tokenburning/internal/platform"
	"github.com/rshatskiy/tokenburning/internal/store"
)

func newConnectCmd() *cobra.Command {
	var to, token, period string
	var breadth, depth, noAutostart bool
	cmd := &cobra.Command{
		Use:   "connect",
		Short: "Подключить локальную установку к команде (сохранить токен, проверить, начать отправку)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if to == "" || token == "" {
				return fmt.Errorf("нужны --to <url> и --token <T> (выдаются на /install сервера)")
			}
			var cats []string
			if breadth {
				cats = append(cats, "breadth")
			}
			if depth {
				cats = append(cats, "depth")
			}
			if len(cats) == 0 {
				cats = []string{"breadth"} // дефолт
			}
			// 1. сохранить конфиг
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			cfg.Push = config.PushCfg{Enabled: true, Categories: cats, Endpoint: to, Token: token}
			if err := config.Save(cfg); err != nil {
				return err
			}
			// 2. мгновенный пробный push (валидирует токен + шлёт первый агрегат)
			home, err := os.UserHomeDir()
			if err != nil {
				return err
			}
			db, err := store.Open(filepath.Join(home, ".tokenburning", "tokenburning.db"))
			if err != nil {
				return err
			}
			defer db.Close()
			payload, err := aggregate.Build(db, cats, period)
			if err != nil {
				return err
			}
			if err := aggregate.Push(payload, to, token); err != nil {
				return fmt.Errorf("проверка подключения не прошла (конфиг сохранён): %w", err)
			}
			cmd.Printf("подключено к %s — отправлено %v\n", to, cats)
			// 3. автозапуск фонового сбора
			if !noAutostart {
				exe, err := os.Executable()
				if err == nil {
					if err := platform.EnableAutostart(exe); err != nil {
						cmd.Printf("предупреждение: не удалось включить автозапуск: %v\n", err)
					} else {
						cmd.Println("фоновый сбор включён (автозапуск при логине)")
					}
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&to, "to", "", "URL сервера команды")
	cmd.Flags().StringVar(&token, "token", "", "персональный токен с /install")
	cmd.Flags().BoolVar(&breadth, "breadth", false, "слать breadth-агрегаты (дефолт, если ничего не выбрано)")
	cmd.Flags().BoolVar(&depth, "depth", false, "слать depth-сигналы сессий")
	cmd.Flags().StringVar(&period, "period", "30d", "период агрегата: 7d|30d|90d|all")
	cmd.Flags().BoolVar(&noAutostart, "no-autostart", false, "не включать автозапуск (только настроить и проверить)")
	return cmd
}
