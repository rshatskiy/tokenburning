package cli

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/rshatskiy/tokenburning/internal/platform"
	"github.com/rshatskiy/tokenburning/internal/store"
	"github.com/rshatskiy/tokenburning/internal/view"
)

func newDashboardCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "dashboard",
		Short: "Открыть локальный web-дашборд",
		RunE: func(cmd *cobra.Command, args []string) error {
			home, err := os.UserHomeDir()
			if err != nil {
				return err
			}
			dbPath := filepath.Join(home, ".tokenburning", "tokenburning.db")
			db, err := store.Open(dbPath)
			if err != nil {
				return err
			}
			defer db.Close()

			tok := make([]byte, 16)
			if _, err := rand.Read(tok); err != nil {
				return err
			}
			token := hex.EncodeToString(tok)

			ln, err := net.Listen("tcp", "127.0.0.1:0") // случайный порт, строго loopback
			if err != nil {
				return err
			}
			url := fmt.Sprintf("http://%s/?t=%s", ln.Addr().String(), token)
			cmd.Printf("tokenburning dashboard: %s\n", url)
			if err := platform.OpenBrowser(url); err != nil {
				cmd.Printf("(не удалось открыть браузер автоматически — откройте ссылку вручную)\n")
			}
			srv := view.NewServer(db, token)
			return http.Serve(ln, srv.Handler())
		},
	}
}
