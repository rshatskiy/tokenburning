package cli

import (
	"github.com/spf13/cobra"

	"github.com/rshatskiy/tokenburning/internal/selfupdate"
)

func newUpdateCmd() *cobra.Command {
	var checkOnly bool
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Обновить tokenburning до последней версии",
		RunE: func(cmd *cobra.Command, _ []string) error {
			tag, err := selfupdate.LatestTag()
			if err != nil {
				return err
			}
			if !selfupdate.IsOlder(version, tag) {
				cmd.Printf("tokenburning %s — уже последняя версия\n", version)
				return nil
			}
			cmd.Printf("доступна новая версия: %s (текущая %s)\n", tag, version)
			if checkOnly {
				cmd.Println("запусти: tokenburning update")
				return nil
			}
			cmd.Println("скачиваю и проверяю контрольную сумму…")
			if err := selfupdate.DownloadAndApply(tag); err != nil {
				return err
			}
			cmd.Printf("обновлено до %s ✓\n", tag)
			return nil
		},
	}
	cmd.Flags().BoolVar(&checkOnly, "check", false, "только проверить наличие новой версии, не устанавливать")
	return cmd
}
