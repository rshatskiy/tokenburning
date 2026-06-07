package cli

import "github.com/spf13/cobra"

var version = "dev"

// NewRootCmd собирает корневую команду tokenburning со всеми подкомандами.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "tokenburning",
		Short: "Единый локальный вид по всем твоим ИИ-инструментам",
	}
	root.AddCommand(newScanCmd())
	root.AddCommand(newDashboardCmd())
	root.AddCommand(newVersionCmd())
	return root
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Показать версию",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Println("tokenburning " + version)
			return nil
		},
	}
}

// Execute — точка входа из main.
func Execute() error {
	return NewRootCmd().Execute()
}
