package cli

import (
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/rshatskiy/tokenburning/internal/insights"
	"github.com/rshatskiy/tokenburning/internal/store"
)

// newOptimizeCmd — поиск «утечек»: дорогие сессии-выбросы, падение кэш-хита,
// неоценённые модели, раздутый CLAUDE.md, лишние MCP.
func newOptimizeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "optimize",
		Short: "Найти, где утекают токены и деньги, и что с этим сделать",
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
			home, _ := os.UserHomeDir()
			list := insights.Build(db, home, time.Now())
			if len(list) == 0 {
				cmd.Println("явных утечек не видно: кэш стабилен, сессии в норме, все модели оценены ✓")
				return nil
			}
			for _, in := range list {
				mark := "•"
				if in.Severity == "warn" {
					mark = "!"
				}
				cmd.Printf("%s %s\n", mark, in.Text)
			}
			return nil
		},
	}
}
