//go:build darwin

package platform

import (
	"strings"
	"testing"
)

func TestPlistContent(t *testing.T) {
	c := plistContent("/opt/tokenburning", "/home/u/.tokenburning/daemon.log")
	for _, want := range []string{autostartLabel, "/opt/tokenburning", "<string>daemon</string>", "RunAtLoad", "daemon.log"} {
		if !strings.Contains(c, want) {
			t.Fatalf("plist missing %q:\n%s", want, c)
		}
	}
}
