package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rshatskiy/tokenburning/internal/config"
)

// planPresets — публично заявленные цены подписок (на июнь 2026).
var planPresets = map[string]float64{
	"claude-pro":    20,
	"claude-max-5x": 100,
	"claude-max":    200,
	"cursor-pro":    20,
	"chatgpt-pro":   200,
}

func newPlanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Подписка для метрики «извлечено из подписки» (дашборд и status)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if cfg.Plan.MonthlyUSD <= 0 {
				cmd.Println("подписка не задана — пример: tokenburning plan set claude-max")
				return nil
			}
			cmd.Printf("подписка: %s — $%.0f/мес\n", cfg.Plan.Preset, cfg.Plan.MonthlyUSD)
			return nil
		},
	}

	var monthly float64
	set := &cobra.Command{
		Use:   "set <claude-pro|claude-max-5x|claude-max|cursor-pro|chatgpt-pro|custom>",
		Short: "Задать подписку (custom — с --monthly-usd)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			preset := args[0]
			usd, ok := planPresets[preset]
			switch {
			case ok:
			case preset == "custom" && monthly > 0:
				usd = monthly
			case preset == "custom":
				return fmt.Errorf("для custom укажите --monthly-usd <сумма>")
			default:
				return fmt.Errorf("неизвестный пресет %q (доступно: claude-pro, claude-max-5x, claude-max, cursor-pro, chatgpt-pro, custom)", preset)
			}
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			cfg.Plan = config.PlanCfg{Preset: preset, MonthlyUSD: usd}
			if err := config.Save(cfg); err != nil {
				return err
			}
			cmd.Printf("подписка: %s — $%.0f/мес. Дашборд покажет «извлечено ×N».\n", preset, usd)
			return nil
		},
	}
	set.Flags().Float64Var(&monthly, "monthly-usd", 0, "цена подписки $/мес (для custom)")

	reset := &cobra.Command{
		Use:   "reset",
		Short: "Убрать подписку",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			cfg.Plan = config.PlanCfg{}
			if err := config.Save(cfg); err != nil {
				return err
			}
			cmd.Println("подписка убрана")
			return nil
		},
	}
	cmd.AddCommand(set, reset)
	return cmd
}

func newAliasCmd() *cobra.Command {
	var list bool
	var remove string
	cmd := &cobra.Command{
		Use:   "alias [<имя-из-логов> <каноническое-имя>]",
		Short: "Сопоставить нестандартное имя модели цене (прокси переименовывают модели)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			switch {
			case list:
				if len(cfg.ModelAliases) == 0 {
					cmd.Println("алиасов нет")
				}
				for k, v := range cfg.ModelAliases {
					cmd.Printf("%s → %s\n", k, v)
				}
				return nil
			case remove != "":
				delete(cfg.ModelAliases, remove)
				return config.Save(cfg)
			case len(args) == 2:
				if cfg.ModelAliases == nil {
					cfg.ModelAliases = map[string]string{}
				}
				cfg.ModelAliases[args[0]] = args[1]
				if err := config.Save(cfg); err != nil {
					return err
				}
				cmd.Printf("%s → %s (применится при следующем scan)\n", args[0], args[1])
				return nil
			default:
				return fmt.Errorf("использование: alias <из> <в> | alias --list | alias --remove <из>")
			}
		},
	}
	cmd.Flags().BoolVar(&list, "list", false, "показать алиасы")
	cmd.Flags().StringVar(&remove, "remove", "", "удалить алиас")
	return cmd
}
