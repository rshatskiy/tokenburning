package main

import (
	"fmt"
	"os"

	"github.com/lens/lens/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "lens:", err)
		os.Exit(1)
	}
}
