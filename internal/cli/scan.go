package cli

import "github.com/spf13/cobra"

func newScanCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "scan",
		Short: "Разобрать локальные логи и показать стоимость",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Println("scan: not implemented yet")
			return nil
		},
	}
}
