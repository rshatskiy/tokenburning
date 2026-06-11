package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rshatskiy/tokenburning/internal/config"
	"github.com/rshatskiy/tokenburning/internal/pricing"
)

// newCurrencyCmd — валюта отображения (хранится всё в USD; курс — ЦБ РФ, кэш сутки).
func newCurrencyCmd() *cobra.Command {
	var reset bool
	cmd := &cobra.Command{
		Use:   "currency [RUB|EUR|USD]",
		Short: "Валюта отображения (курс ЦБ РФ, обновляется раз в сутки)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if reset || (len(args) == 1 && strings.EqualFold(args[0], "USD")) {
				cfg.Currency = ""
				if err := config.Save(cfg); err != nil {
					return err
				}
				cmd.Println("валюта: USD")
				return nil
			}
			if len(args) == 0 {
				cur := cfg.Currency
				if cur == "" {
					cur = "USD"
				}
				cmd.Printf("валюта: %s\n", cur)
				return nil
			}
			code := strings.ToUpper(args[0])
			rate, err := pricing.FXRate(code)
			if err != nil {
				return err
			}
			cfg.Currency = code
			if err := config.Save(cfg); err != nil {
				return err
			}
			cmd.Printf("валюта: %s (курс ЦБ: 1$ = %.2f %s)\n", code, rate, pricing.CurrencySymbol(code))
			return nil
		},
	}
	cmd.Flags().BoolVar(&reset, "reset", false, "вернуть USD")
	return cmd
}

// moneyFmt возвращает форматтер сумм в валюте пользователя (fallback — USD).
func moneyFmt() func(usd float64) string {
	cfg, err := config.Load()
	if err == nil && cfg.Currency != "" && !strings.EqualFold(cfg.Currency, "USD") {
		if rate, ferr := pricing.FXRate(cfg.Currency); ferr == nil && rate > 0 {
			sym := pricing.CurrencySymbol(cfg.Currency)
			return func(usd float64) string { return fmt.Sprintf("%.2f %s", usd*rate, sym) }
		}
	}
	return func(usd float64) string { return fmt.Sprintf("$%.2f", usd) }
}
