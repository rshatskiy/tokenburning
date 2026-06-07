package cli

import "github.com/spf13/cobra"

// NewRootCmd собирает корневую команду lens со всеми подкомандами.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "lens",
		Short: "Единый локальный вид по всем твоим ИИ-инструментам",
	}
	root.AddCommand(newScanCmd())
	return root
}

// Execute — точка входа из main.
func Execute() error {
	return NewRootCmd().Execute()
}
