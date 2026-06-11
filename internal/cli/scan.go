package cli

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/rshatskiy/tokenburning/internal/aggregate"
	"github.com/rshatskiy/tokenburning/internal/collect"
	"github.com/rshatskiy/tokenburning/internal/config"
	"github.com/rshatskiy/tokenburning/internal/platform"
	"github.com/rshatskiy/tokenburning/internal/pricing"
	"github.com/rshatskiy/tokenburning/internal/store"
)

// scanOpts — период, фильтры и формат вывода сводки.
type scanOpts struct {
	Period   string // today|7d|30d|90d|month|all
	From, To string // YYYY-MM-DD (перекрывают Period)
	Project  string
	Tool     string
	Format   string // table|json|csv
}

func newScanCmd() *cobra.Command {
	var o scanOpts
	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Разобрать локальные логи и показать стоимость",
		RunE: func(cmd *cobra.Command, args []string) error {
			dbPath, err := store.DefaultPath()
			if err != nil {
				return err
			}
			out, err := runScan(dbPath, o)
			if err != nil {
				return err
			}
			cmd.Print(out)
			return nil
		},
	}
	cmd.Flags().StringVar(&o.Period, "period", "all", "период: today|7d|30d|90d|month|all")
	cmd.Flags().StringVar(&o.From, "from", "", "начало периода YYYY-MM-DD")
	cmd.Flags().StringVar(&o.To, "to", "", "конец периода YYYY-MM-DD (исключительно)")
	cmd.Flags().StringVar(&o.Project, "project", "", "фильтр по подстроке проекта")
	cmd.Flags().StringVar(&o.Tool, "tool", "", "фильтр по инструменту (claude_code, codex, ...)")
	cmd.Flags().StringVar(&o.Format, "format", "table", "формат: table|json|csv")
	return cmd
}

// filterFor превращает opts в store.Filter (--from/--to важнее --period).
func filterFor(o scanOpts, now time.Time) (store.Filter, error) {
	f := store.Filter{Project: o.Project, Tool: o.Tool}
	if o.From != "" || o.To != "" {
		if o.From != "" {
			t, err := time.ParseInLocation("2006-01-02", o.From, time.Local)
			if err != nil {
				return f, fmt.Errorf("--from: %w", err)
			}
			f.Since = t
		}
		if o.To != "" {
			t, err := time.ParseInLocation("2006-01-02", o.To, time.Local)
			if err != nil {
				return f, fmt.Errorf("--to: %w", err)
			}
			f.Until = t
		}
		if !f.Since.IsZero() && !f.Until.IsZero() && f.Until.Before(f.Since) {
			return f, fmt.Errorf("--to раньше --from")
		}
		return f, nil
	}
	switch o.Period {
	case "", "all":
	case "today":
		f.Since = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	case "month":
		f.Since = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	case "7d", "30d", "90d":
		days := map[string]int{"7d": 7, "30d": 30, "90d": 90}[o.Period]
		f.Since = aggregate.SinceForDays(days, now)
	default:
		return f, fmt.Errorf("неизвестный период %q (today|7d|30d|90d|month|all)", o.Period)
	}
	return f, nil
}

// collectInto: discover → collect → price → store (общий проход scan/status/dashboard).
func collectInto(db *store.DB) (collect.Result, error) {
	paths, err := platform.Detect()
	if err != nil {
		return collect.Result{}, err
	}
	cfg, _ := config.Load()
	_ = pricing.RefreshLive(3 * time.Second) // best-effort: офлайн — работаем на снапшоте
	cat, err := pricing.LoadEffective(cfg.ModelAliases)
	if err != nil {
		return collect.Result{}, err
	}
	return collect.Run(db, cat, paths, func(tool string, i, n int) {
		fmt.Fprintf(os.Stderr, "\r%s %d/%d…", tool, i, n)
	})
}

// runScan: сбор + сводка в выбранном формате.
func runScan(dbPath string, o scanOpts) (string, error) {
	db, err := store.Open(dbPath)
	if err != nil {
		return "", err
	}
	defer db.Close()

	res, err := collectInto(db)
	if err != nil {
		return "", err
	}
	fmt.Fprintln(os.Stderr)

	f, err := filterFor(o, time.Now())
	if err != nil {
		return "", err
	}
	tools, err := db.FilteredByTool(f)
	if err != nil {
		return "", err
	}
	models, err := db.FilteredByModel(f)
	if err != nil {
		return "", err
	}
	total, err := db.CostTotal(f)
	if err != nil {
		return "", err
	}

	switch o.Format {
	case "json":
		b, err := json.MarshalIndent(map[string]any{
			"period": o.Period, "from": o.From, "to": o.To,
			"project": o.Project, "tool": o.Tool,
			"totalCost": total, "byTool": tools, "byModel": models,
		}, "", "  ")
		return string(b) + "\n", err
	case "csv":
		var b strings.Builder
		w := csv.NewWriter(&b)
		_ = w.Write([]string{"kind", "name", "events", "tokens", "cost_usd"})
		for _, t := range tools {
			_ = w.Write([]string{"tool", t.Tool, fmt.Sprint(t.Events), fmt.Sprint(t.Tokens), fmt.Sprintf("%.4f", t.CostAmount)})
		}
		for _, m := range models {
			_ = w.Write([]string{"model", m.Model, fmt.Sprint(m.Events), fmt.Sprint(m.Tokens), fmt.Sprintf("%.4f", m.CostAmount)})
		}
		w.Flush()
		return b.String(), w.Error()
	case "", "table":
		var b strings.Builder
		fmt.Fprintf(&b, "%-14s %8s %14s %12s\n", "TOOL", "EVENTS", "TOKENS", "COST(USD)")
		for _, t := range tools {
			fmt.Fprintf(&b, "%-14s %8d %14d %12.1f\n", t.Tool, t.Events, t.Tokens, t.CostAmount)
		}
		fmt.Fprintf(&b, "\n%-26s %8s %12s\n", "MODEL", "EVENTS", "COST(USD)")
		for _, m := range models {
			fmt.Fprintf(&b, "%-26s %8d %12.1f\n", m.Model, m.Events, m.CostAmount)
		}
		if res.Quarantined > 0 {
			fmt.Fprintf(&b, "\nв карантине записей: %d\n", res.Quarantined)
			for _, e := range res.SampleErrors {
				fmt.Fprintf(&b, "  пример: %s\n", e)
			}
		}
		return b.String(), nil
	default:
		return "", fmt.Errorf("неизвестный формат %q (table|json|csv)", o.Format)
	}
}
