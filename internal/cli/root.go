package cli

import "github.com/spf13/cobra"

// NewRootCmd собирает корневую команду tokenburning со всеми подкомандами.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "tokenburning",
		Short: "Единый локальный вид по всем твоим ИИ-инструментам",
	}
	root.AddCommand(newScanCmd())
	root.AddCommand(newDashboardCmd())
	return root
}

// Execute — точка входа из main.
func Execute() error {
	return NewRootCmd().Execute()
}
