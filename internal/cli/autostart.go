package cli

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/rshatskiy/tokenburning/internal/config"
	"github.com/rshatskiy/tokenburning/internal/platform"
)

func newEnableCmd() *cobra.Command {
	var intervalMin int
	var to, token string
	var breadth, depth bool
	cmd := &cobra.Command{
		Use:   "enable",
		Short: "Включить фоновый сбор (автозапуск демона)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if intervalMin > 0 {
				cfg.IntervalMinutes = intervalMin
			}
			// настроить push по согласию, если заданы endpoint и категории
			var cats []string
			if breadth {
				cats = append(cats, "breadth")
			}
			if depth {
				cats = append(cats, "depth")
			}
			if to != "" && len(cats) > 0 {
				cfg.Push = config.PushCfg{Enabled: true, Categories: cats, Endpoint: to, Token: token}
			}
			if err := config.Save(cfg); err != nil {
				return err
			}
			exe, err := os.Executable()
			if err != nil {
				return err
			}
			if err := platform.EnableAutostart(exe); err != nil {
				return err
			}
			ok, where := platform.AutostartInstalled()
			cmd.Printf("фоновый сбор включён (%v): %s\n", ok, where)
			if cfg.Push.Enabled {
				cmd.Printf("push наверх включён: %v → %s\n", cfg.Push.Categories, cfg.Push.Endpoint)
			} else {
				cmd.Println("push наверх не настроен (только локальный сбор). Чтобы включить: --to <url> --breadth/--depth")
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&intervalMin, "interval-min", 0, "интервал сбора в минутах (по умолчанию 15)")
	cmd.Flags().StringVar(&to, "to", "", "URL сервера для отправки агрегатов (опционально)")
	cmd.Flags().StringVar(&token, "token", "", "токен коллектора для отправки")
	cmd.Flags().BoolVar(&breadth, "breadth", false, "слать breadth-агрегаты (с --to)")
	cmd.Flags().BoolVar(&depth, "depth", false, "слать depth-сигналы (с --to)")
	return cmd
}

func newDisableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "disable",
		Short: "Выключить фоновый сбор (снять автозапуск)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := platform.DisableAutostart(); err != nil {
				return err
			}
			// выключить push в конфиге, конфиг оставить
			if cfg, err := config.Load(); err == nil {
				cfg.Push.Enabled = false
				_ = config.Save(cfg)
			}
			cmd.Println("фоновый сбор выключен")
			return nil
		},
	}
}
