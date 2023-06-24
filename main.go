package main

import (
	"log"
	"os"

	"github.com/urfave/cli/v2"
)

func Run(*cli.Context) error {

	brif := newBrif()
	_ = brif
	return nil
}

// https://cli.urfave.org/v2/getting-started/
func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	app := &cli.App{
		Name:   "brif",
		Usage:  "summarize novel ",
		Action: Run,
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
