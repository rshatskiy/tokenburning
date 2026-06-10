package cli

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/rshatskiy/tokenburning/internal/collect"
	"github.com/rshatskiy/tokenburning/internal/platform"
	"github.com/rshatskiy/tokenburning/internal/pricing"
	"github.com/rshatskiy/tokenburning/internal/store"
)

func newScanCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "scan",
		Short: "Разобрать локальные логи и показать стоимость",
		RunE: func(cmd *cobra.Command, args []string) error {
			dbPath, err := store.DefaultPath()
			if err != nil {
				return err
			}
			out, err := runScan(dbPath)
			if err != nil {
				return err
			}
			cmd.Print(out)
			return nil
		},
	}
}

// runScan: discover → collect → price → store → summary. Возвращает текст таблицы.
func runScan(dbPath string) (string, error) {
	paths, err := platform.Detect()
	if err != nil {
		return "", err
	}
	cat, err := pricing.LoadEmbedded()
	if err != nil {
		return "", err
	}
	db, err := store.Open(dbPath)
	if err != nil {
		return "", err
	}
	defer db.Close()

	// TODO slice-2: стримить инжест чанками, если корпус вырастет за ~100k событий (сейчас весь batch в памяти).
	res, err := collect.Run(db, cat, paths, func(tool string, i, n int) {
		fmt.Fprintf(os.Stderr, "\r%s %d/%d…", tool, i, n)
	})
	if err != nil {
		return "", err
	}
	fmt.Fprintln(os.Stderr)
	quarantined := res.Quarantined

	var b strings.Builder
	tools, err := db.SummaryByTool(time.Time{})
	if err != nil {
		return "", err
	}
	fmt.Fprintf(&b, "%-14s %8s %14s %12s\n", "TOOL", "EVENTS", "TOKENS", "COST(USD)")
	for _, tsum := range tools {
		fmt.Fprintf(&b, "%-14s %8d %14d %12.1f\n", tsum.Tool, tsum.Events, tsum.Tokens, tsum.CostAmount)
	}

	rows, err := db.SummaryByModel(time.Time{})
	if err != nil {
		return "", err
	}
	fmt.Fprintf(&b, "\n%-26s %8s %12s\n", "MODEL", "EVENTS", "COST(USD)")
	for _, r := range rows {
		fmt.Fprintf(&b, "%-26s %8d %12.1f\n", r.Model, r.Events, r.CostAmount)
	}
	if quarantined > 0 {
		fmt.Fprintf(&b, "\nв карантине записей: %d\n", quarantined)
		for _, e := range res.SampleErrors {
			fmt.Fprintf(&b, "  пример: %s\n", e)
		}
	}
	return b.String(), nil
}
