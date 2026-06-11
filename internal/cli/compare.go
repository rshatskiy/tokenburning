package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/rshatskiy/tokenburning/internal/quality"
	"github.com/rshatskiy/tokenburning/internal/store"
)

// newCompareCmd — сравнение моделей по качеству: one-shot rate и retry
// из tool_use-блоков локальных логов (пока только Claude Code — у него
// есть file-level данные).
func newCompareCmd() *cobra.Command {
	var period string
	cmd := &cobra.Command{
		Use:   "compare",
		Short: "Сравнить модели: one-shot rate, retry, стоимость",
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
			f, err := filterFor(scanOpts{Period: period}, time.Now())
			if err != nil {
				return err
			}
			rows, err := db.RawToolEvents(f)
			if err != nil {
				return err
			}
			qs := quality.Compute(rows)
			if len(qs) == 0 {
				cmd.Println("нет данных с правками файлов за период (метрика пока считается по Claude Code)")
				return nil
			}
			costs := map[string]float64{}
			if ms, merr := db.FilteredByModel(f); merr == nil {
				for _, m := range ms {
					costs[m.Model] = m.CostAmount
				}
			}
			var b strings.Builder
			fmt.Fprintf(&b, "%-26s %8s %8s %9s %10s %12s\n", "MODEL", "EDITS", "RETRIES", "ONE-SHOT", "SESSIONS", "COST(USD)")
			for _, q := range qs {
				fmt.Fprintf(&b, "%-26s %8d %8d %8.0f%% %10d %12.1f\n",
					q.Model, q.EditTurns, q.Retries, q.OneShotPct, q.Sessions, costs[q.Model])
			}
			b.WriteString("\none-shot = правка принята без повторного редактирования того же файла после shell-команды; оценка по локальным логам\n")
			cmd.Print(b.String())
			return nil
		},
	}
	cmd.Flags().StringVar(&period, "period", "30d", "период: today|7d|30d|90d|month|all")
	return cmd
}
