package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

const (
	VersionMaj   = 0
	VersionMin   = 1
	VersionPatch = 0
)

func numPlaces(n int) int {
	if n == 0 {
		return 1
	}
	return int(math.Log10(math.Abs(float64(n)))) + 1
}

func main() {
	app := cli.NewApp()
	app.Name = "ycopy"
	app.Version = fmt.Sprintf("%d.%d.%d", VersionMaj, VersionMin, VersionPatch)
	app.Usage = "copy files from a list"
	app.Flags = []cli.Flag{
		cli.StringFlag{Name: "list-file, l", Required: true},
		cli.StringFlag{Name: "source-path, s"},
		cli.StringFlag{Name: "target-path, t"},
		cli.BoolFlag{Name: "stop-on-first-error, e"},
		cli.BoolFlag{Name: "debug, d"},
	}
	app.Action = func(c *cli.Context) error {
		list := c.String("list-file")

		source, err := filepath.Abs(c.String("source-path"))
		if err != nil {
			return errors.Wrap(err, "error in source-path")
		}

		target, err := filepath.Abs(c.String("target-path"))
		if err != nil {
			return errors.Wrap(err, "error in target-path")
		}

		copiers, err := MakeCopiers(list, source, target)
		if err != nil {
			return errors.Wrapf(err, "error in %s", list)
		}
		cfg := &Config{
			StopOnFirstError: c.Bool("stop-on-first-error"),
			Copiers:          copiers,
		}

		if c.Bool("debug") {
			fmt.Printf("Stop on first error: %v\n", cfg.StopOnFirstError)
			fmt.Printf("Operations:\n")
			padfmt := fmt.Sprintf(" %%%dd: %%s\n", numPlaces(len(cfg.Copiers)))
			for i, c := range cfg.Copiers {
				fmt.Printf(padfmt, i, c.DisplayIntent())
			}
			return nil
		}

		log.Printf("Starting %d operations...", len(cfg.Copiers))
		for i, c := range cfg.Copiers {
			msg, err := c.Copy()
			if err != nil {
				if cfg.StopOnFirstError {
					return err
				}
				msg = fmt.Sprintf("ERROR: %v (file from %s line %d)", err, list, c.Context().Line)
			}
			log.Printf(" %d: %s", i+1, msg)
		}

		log.Printf("Done")
		return nil
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
