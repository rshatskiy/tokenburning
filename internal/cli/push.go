package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/rshatskiy/tokenburning/internal/aggregate"
	"github.com/rshatskiy/tokenburning/internal/store"
)

func newPushCmd() *cobra.Command {
	var breadth, depth, dryRun bool
	var to, period, token string
	cmd := &cobra.Command{
		Use:   "push",
		Short: "Отдать наверх агрегаты по согласию (только производные; ничего сырого не уходит)",
		RunE: func(cmd *cobra.Command, args []string) error {
			var cats []string
			if breadth {
				cats = append(cats, "breadth")
			}
			if depth {
				cats = append(cats, "depth")
			}
			if len(cats) == 0 {
				return fmt.Errorf("укажите хотя бы одну категорию согласия: --breadth и/или --depth")
			}
			switch period {
			case "7d", "30d", "90d", "all":
			default:
				return fmt.Errorf("неизвестный период %q (допустимо: 7d, 30d, 90d, all)", period)
			}
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
			data, err := json.MarshalIndent(payload, "", "  ")
			if err != nil {
				return err
			}
			if dryRun {
				cmd.Println(string(data))
				return nil
			}
			if to == "" {
				to = os.Getenv("TOKENBURNING_SERVER")
			}
			if to == "" {
				return fmt.Errorf("сервер не задан: --to <url> или TOKENBURNING_SERVER (приёмник — отдельный спек; пока используйте --dry-run)")
			}
			if token == "" {
				token = os.Getenv("TOKENBURNING_TOKEN")
			}
			if err := aggregate.Push(payload, to, token); err != nil {
				return err
			}
			cmd.Printf("отправлено наверх: %v (%d байт)\n", cats, len(data))
			return nil
		},
	}
	cmd.Flags().BoolVar(&breadth, "breadth", false, "включить breadth-агрегаты (стоимость/активность/инструменты/модели)")
	cmd.Flags().BoolVar(&depth, "depth", false, "включить depth-сигналы сессий (медианы)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "напечатать точный payload, ничего не отправляя")
	cmd.Flags().StringVar(&to, "to", "", "URL сервера-приёмника (или env TOKENBURNING_SERVER)")
	cmd.Flags().StringVar(&period, "period", "30d", "период: 7d|30d|90d|all")
	cmd.Flags().StringVar(&token, "token", "", "токен коллектора (выдаётся на /install сервера)")
	return cmd
}
