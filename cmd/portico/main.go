package main

import (
	"os"

	"github.com/catielanier/portico/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
