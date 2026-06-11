package cli

import (
	"os"

	"github.com/spf13/cobra"
)

var version = "dev"

// NewRootCmd собирает корневую команду tokenburning со всеми подкомандами.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "tokenburning",
		Short: "Единый локальный вид по всем твоим ИИ-инструментам",
	}
	// Без явного SetOut cobra шлёт cmd.Print* в stderr (fallback) —
	// `tokenburning scan > file` давал бы пустой файл.
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	root.AddCommand(newScanCmd())
	root.AddCommand(newDashboardCmd())
	root.AddCommand(newVersionCmd())
	root.AddCommand(newPushCmd())
	root.AddCommand(newDaemonCmd())
	root.AddCommand(newEnableCmd())
	root.AddCommand(newDisableCmd())
	root.AddCommand(newConnectCmd())
	root.AddCommand(newUpdateCmd())
	root.AddCommand(newStatusCmd())
	root.AddCommand(newOptimizeCmd())
	root.AddCommand(newCurrencyCmd())
	root.AddCommand(newPlanCmd())
	root.AddCommand(newAliasCmd())
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
