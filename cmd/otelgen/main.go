package main

import (
	"log"
	"os"

	"github.com/krzko/otelgen/internal/cli"
)

var (
	version string
	commit  string
	date    string
)

func main() {
	app := cli.New(version, commit, date)

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
