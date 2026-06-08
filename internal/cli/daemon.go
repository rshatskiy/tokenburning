package cli

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/rshatskiy/tokenburning/internal/config"
	"github.com/rshatskiy/tokenburning/internal/daemon"
)

func newDaemonCmd() *cobra.Command {
	var interval time.Duration
	cmd := &cobra.Command{
		Use:   "daemon",
		Short: "Фоновый периодический сбор (обычно запускается автозапуском)",
		RunE: func(cmd *cobra.Command, args []string) error {
			home, err := os.UserHomeDir()
			if err != nil {
				return err
			}
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			iv := interval
			if iv <= 0 {
				iv = cfg.Interval()
			}
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()
			cmd.Printf("tokenburning daemon: интервал %s\n", iv)
			return daemon.Run(ctx, daemon.Options{
				DBPath:         filepath.Join(home, ".tokenburning", "tokenburning.db"),
				Interval:       iv,
				PushEnabled:    cfg.Push.Enabled,
				PushCategories: cfg.Push.Categories,
				PushEndpoint:   cfg.Push.Endpoint,
				PushToken:      cfg.Push.Token,
				AutoUpdate:     cfg.AutoUpdate,
				Version:        version,
				Log:            func(s string) { cmd.Println(s) },
			})
		},
	}
	cmd.Flags().DurationVar(&interval, "interval", 0, "интервал сбора (по умолчанию из конфига или 15m)")
	return cmd
}
