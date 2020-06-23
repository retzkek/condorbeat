package main

import (
	"os"

	"github.com/retzkek/condorbeat/cmd"

	_ "github.com/retzkek/condorbeat/include"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
