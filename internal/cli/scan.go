package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/lens/lens/internal/adapter"
	"github.com/lens/lens/internal/adapter/claudecode"
	"github.com/lens/lens/internal/model"
	"github.com/lens/lens/internal/platform"
	"github.com/lens/lens/internal/pricing"
	"github.com/lens/lens/internal/store"
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
			dbPath := filepath.Join(home, ".lens", "lens.db")
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

	ad := claudecode.New()
	sources, err := ad.Discover(paths)
	if err != nil {
		return "", err
	}

	// TODO slice-2: стримить инжест чанками, если корпус вырастет за ~100k событий (сейчас весь batch в памяти).
	var batch []model.Event
	var quarantined int
	emit := func(e model.Event) {
		e.Cost = cat.Cost(e.Model, e.Tokens)
		batch = append(batch, e)
	}
	quar := func(raw []byte, err error) { quarantined++ }

	for i, src := range sources {
		// TODO slice-2: подавлять прогресс при не-TTY stderr (isatty), сейчас \r может засорять перенаправленный вывод.
		fmt.Fprintf(os.Stderr, "\rскан %d/%d…", i+1, len(sources)) // прогресс первого прохода (§8.4)
		if _, err := ad.Collect(src, adapter.Cursor{}, emit, quar); err != nil {
			fmt.Fprintf(os.Stderr, "\nпропуск %s: %v\n", src.Path, err)
		}
	}
	fmt.Fprintln(os.Stderr)
	if err := db.Insert(batch); err != nil {
		return "", err
	}

	rows, err := db.SummaryByModel()
	if err != nil {
		return "", err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%-22s %8s %12s\n", "MODEL", "EVENTS", "COST(USD)")
	for _, r := range rows {
		fmt.Fprintf(&b, "%-22s %8d %12.1f\n", r.Model, r.Events, r.CostAmount)
	}
	if quarantined > 0 {
		fmt.Fprintf(&b, "\nв карантине записей: %d\n", quarantined)
	}
	return b.String(), nil
}
