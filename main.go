package main

import (
	"os"

	"github.com/elastic/beats/libbeat/beat"

	"github.com/retzkek/condorbeat/beater"
)

func main() {
	err := beat.Run("condorbeat", "", beater.New)
	if err != nil {
		os.Exit(1)
	}
}
