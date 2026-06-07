package main

import (
	"fmt"
	"os"

	"github.com/rshatskiy/tokenburning/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "tokenburning:", err)
		os.Exit(1)
	}
}
