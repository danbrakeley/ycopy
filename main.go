package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"

	"github.com/danbrakeley/ycopy/copier"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

const (
	VersionMaj   = 0
	VersionMin   = 2
	VersionPatch = 1
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
	app.Usage = "copy (and/or download) files based on a newline-separated list in a text file"
	app.ArgsUsage = "<list-file>"
	app.HideHelp = true
	app.Flags = []cli.Flag{
		cli.BoolFlag{Name: "help, h, ?", Usage: "show this message"},
		cli.StringFlag{Name: "src", Usage: "look for local files under this path (default: current working directory)"},
		cli.StringFlag{Name: "dest", Usage: "write files/folders under this path (default: current working directory)"},
		cli.BoolFlag{Name: "dryrun, d", Usage: "preview work (without doing work)"},
		cli.IntFlag{Name: "threads", Value: 1, Usage: "number of operations to perform in parallel"},
		cli.BoolFlag{Name: "verbose", Usage: "add extra logging (for debugging)"},
	}
	app.Action = func(c *cli.Context) error {
		if c.Bool("help") {
			cli.ShowAppHelpAndExit(c, 0)
		}
		args := c.Args()
		// TODO: make this better
		if len(args) > 1 {
			fmt.Fprintf(os.Stderr, "Unexpected \"%s\" (options go before <list-file>, see --help)\n", strings.Join(args[1:], " "))
			os.Exit(1)
		}
		list := strings.TrimSpace(args.Get(0))
		if len(list) == 0 {
			fmt.Fprintf(os.Stderr, "Argument <list-file> is missing, but is required.\n\n")
			cli.ShowAppHelpAndExit(c, 1)
		}

		src, err := filepath.Abs(c.String("src"))
		if err != nil {
			return errors.Wrap(err, "invalid src")
		}

		dest, err := filepath.Abs(c.String("dest"))
		if err != nil {
			return errors.Wrap(err, "invalid dest")
		}

		copiers, err := MakeCopiers(list, src, dest)
		if err != nil {
			return errors.Wrap(err, "invalid list file")
		}

		cfg := &Config{
			Threads: c.Int("threads"),
			DryRun:  c.Bool("dryrun"),
			Verbose: c.Bool("verbose"),
			Copiers: copiers,
		}

		if cfg.DryRun {
			fmt.Printf("Threads: %d\n", cfg.Threads)
			fmt.Printf("Operations:\n")
			padfmt := fmt.Sprintf(" %%%dd: %%s\n", numPlaces(len(cfg.Copiers)))
			for i, c := range cfg.Copiers {
				fmt.Printf(padfmt, i, c.DisplayIntent())
			}
			return nil
		}

		// install signal hanlder
		chSignal := make(chan os.Signal, 1)
		signal.Notify(chSignal, os.Interrupt)
		signalHandler := func(s os.Signal) {
			if cfg.Verbose {
				log.Printf("----- got signal: %v", s)
			}
			log.Printf("Interrupt detected, running actions will complete, but new actions will not be started...")
		}

		chCopier := make(chan copier.Copier)
		chResult := make(chan Result)

		var wgCopiers sync.WaitGroup
		wgCopiers.Add(cfg.Threads)
		for i := 0; i < cfg.Threads; i++ {
			go workerCopy(cfg, &wgCopiers, i+1, chCopier, chResult)
		}

		var wgResults sync.WaitGroup
		wgResults.Add(1)
		go workerResults(cfg, &wgResults, chResult)

		log.Printf("Starting %d operations across %d threads...", len(cfg.Copiers), cfg.Threads)
	CopierLoop:
		for _, c := range cfg.Copiers {
			select {
			case chCopier <- c:
			case s := <-chSignal:
				signalHandler(s)
				break CopierLoop
			}
		}

		// we're just waiting to end at this point, so an interrupt can be ignored,
		// but the user doesn't know that, so output a consistent response.
		go func() {
			s := <-chSignal
			signalHandler(s)
		}()

		close(chCopier)
		if cfg.Verbose {
			log.Printf("----- waiting for all copy threads to complete")
		}
		wgCopiers.Wait()
		close(chResult)
		if cfg.Verbose {
			log.Printf("----- waiting for resulits thread to complete")
		}
		wgResults.Wait()

		log.Printf("Done")
		return nil
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

type Result struct {
	ThreadID int
	Err      error
	Msg      string
	Ctx      copier.Context
}

func workerCopy(cfg *Config, wg *sync.WaitGroup, threadID int, chCopier chan copier.Copier, chResult chan Result) {
	defer wg.Done()
	if cfg.Verbose {
		log.Printf("----- workerCopy %d: starting", threadID)
	}
	count := 0
	for {
		c, ok := <-chCopier
		if !ok {
			if cfg.Verbose {
				log.Printf("----- workerCopy %d: closed after %d iterations", threadID, count)
			}
			return
		}
		count++
		msg, err := c.Copy()
		chResult <- Result{ThreadID: threadID, Err: err, Msg: msg, Ctx: c.Context()}
	}
}

func workerResults(cfg *Config, wg *sync.WaitGroup, ch chan Result) {
	defer wg.Done()
	if cfg.Verbose {
		log.Printf("----- workerResults: starting")
	}
	count := 0
	for {
		r, ok := <-ch
		if !ok {
			if cfg.Verbose {
				log.Printf("----- workerResults: closed after %d iterations", count)
			}
			return
		}
		count++
		msg := r.Msg
		if r.Err != nil {
			msg = fmt.Sprintf("ERROR: %v", r.Err)
			if len(r.Msg) == 0 {
				msg += " (" + r.Msg + ")"
			}
		}
		log.Printf(" ThreadID %d; Line %d: %s", r.ThreadID, r.Ctx.Line, msg)
	}
}
