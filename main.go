package main

import (
	"os"

	"github.com/retzkek/condorbeat/cmd"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
