package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/rshatskiy/tokenburning/internal/adapter"
	"github.com/rshatskiy/tokenburning/internal/adapter/claudecode"
	"github.com/rshatskiy/tokenburning/internal/adapter/codex"
	"github.com/rshatskiy/tokenburning/internal/model"
	"github.com/rshatskiy/tokenburning/internal/platform"
	"github.com/rshatskiy/tokenburning/internal/pricing"
	"github.com/rshatskiy/tokenburning/internal/store"
)

func newScanCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "scan",
		Short: "Разобрать локальные логи и показать стоимость",
		RunE: func(cmd *cobra.Command, args []string) error {
			home, err := os.UserHomeDir()
			if err != nil {
				return err
			}
			dbPath := filepath.Join(home, ".tokenburning", "tokenburning.db")
			if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
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

	adapters := []adapter.Adapter{
		claudecode.New(),
		codex.New(),
		// cursor добавляется в Task 5
	}

	// TODO slice-2: стримить инжест чанками, если корпус вырастет за ~100k событий (сейчас весь batch в памяти).
	var batch []model.Event
	var quarantined int
	emit := func(e model.Event) {
		e.Cost = cat.Cost(e.Model, e.Tokens)
		batch = append(batch, e)
	}
	quar := func(raw []byte, err error) { quarantined++ }

	for _, ad := range adapters {
		sources, derr := ad.Discover(paths)
		if derr != nil {
			fmt.Fprintf(os.Stderr, "discover %s: %v\n", ad.Name(), derr)
			continue
		}
		for i, src := range sources {
			// TODO slice-2: подавлять прогресс при не-TTY stderr (isatty), сейчас \r может засорять перенаправленный вывод.
			fmt.Fprintf(os.Stderr, "\r%s %d/%d…", ad.Name(), i+1, len(sources))
			if _, cerr := ad.Collect(src, adapter.Cursor{}, emit, quar); cerr != nil {
				fmt.Fprintf(os.Stderr, "\nпропуск %s: %v\n", src.Path, cerr)
			}
		}
		fmt.Fprintln(os.Stderr)
	}

	if err := db.Insert(batch); err != nil {
		return "", err
	}

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
	}
	return b.String(), nil
}
