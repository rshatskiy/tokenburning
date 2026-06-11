package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/rshatskiy/tokenburning/internal/config"
	"github.com/rshatskiy/tokenburning/internal/store"
)

// newStatusCmd — компактная строка для статус-баров и скриптов:
// «today $X · month $Y» (+ «извлечено ×N», если задана подписка).
func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Одна строка: сегодня и за месяц",
		RunE: func(cmd *cobra.Command, args []string) error {
			dbPath, err := store.DefaultPath()
			if err != nil {
				return err
			}
			db, err := store.Open(dbPath)
			if err != nil {
				return err
			}
			defer db.Close()
			if _, err := collectInto(db); err != nil {
				return err
			}
			now := time.Now()
			today, err := db.CostTotal(store.Filter{Since: time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)})
			if err != nil {
				return err
			}
			month, err := db.CostTotal(store.Filter{Since: time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)})
			if err != nil {
				return err
			}
			money := moneyFmt()
			line := fmt.Sprintf("today %s · month %s", money(today), money(month))
			if cfg, cerr := config.Load(); cerr == nil && cfg.Plan.MonthlyUSD > 0 && month > 0 {
				line += fmt.Sprintf(" · ×%.1f из подписки $%.0f", month/cfg.Plan.MonthlyUSD, cfg.Plan.MonthlyUSD)
			}
			cmd.Println(line)
			return nil
		},
	}
}
