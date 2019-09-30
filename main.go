package main

import (
	"fmt"
	"io"
	"math"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"

	"github.com/danbrakeley/ycopy/copier"
	"github.com/danbrakeley/ycopy/frog"
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
	log := frog.New()
	defer log.Close()
	actualHelpPrinter := cli.HelpPrinter
	cli.HelpPrinter = func(w io.Writer, templ string, data interface{}) {
		actualHelpPrinter(w, templ, data)
		log.Close()
	}

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

		if c.Bool("verbose") {
			log.SetMinLevel(frog.Verbose)
		}

		args := c.Args()
		if len(args) > 1 {
			log.Errorf("Unexpected \"%s\" (options go before <list-file>, see --help)", strings.Join(args[1:], " "))
			cli.ShowAppHelpAndExit(c, 1)
		}
		list := strings.TrimSpace(args.Get(0))
		if len(list) == 0 {
			log.Errorf("Argument <list-file> is missing, but is required.")
			cli.ShowAppHelpAndExit(c, 1)
		}

		src, err := filepath.Abs(c.String("src"))
		if err != nil {
			log.Errorf("option <src>: %v", err)
			cli.ShowAppHelpAndExit(c, 1)
		}

		dest, err := filepath.Abs(c.String("dest"))
		if err != nil {
			log.Errorf("option <dest>: %v", err)
			cli.ShowAppHelpAndExit(c, 1)
		}

		copiers, err := MakeCopiers(list, src, dest)
		if err != nil {
			log.Errorf("%v", err)
			cli.ShowAppHelpAndExit(c, 1)
		}

		cfg := &Config{
			Threads: c.Int("threads"),
			DryRun:  c.Bool("dryrun"),
			Copiers: copiers,
		}

		if cfg.DryRun {
			log.Infof("Threads: %d\n", cfg.Threads)
			log.Infof("Operations:\n")
			padfmt := fmt.Sprintf(" %%%dd: %%s\n", numPlaces(len(cfg.Copiers)))
			for i, c := range cfg.Copiers {
				log.Infof(padfmt, i, c.DebugPrint())
			}
			return nil
		}

		// install signal hanlder
		chSignal := make(chan os.Signal, 1)
		signal.Notify(chSignal, os.Interrupt)
		signalHandler := func(s os.Signal) {
			log.Verbosef("got signal: %v", s)
			log.Warningf("Interrupt detected, running actions will complete, but new actions will not be started...")
		}

		chCopier := make(chan copier.Copier)
		chResult := make(chan Result)

		var wgCopiers sync.WaitGroup
		wgCopiers.Add(cfg.Threads)
		for i := 0; i < cfg.Threads; i++ {
			go workerCopy(log, cfg, &wgCopiers, i+1, chCopier, chResult)
		}

		var wgResults sync.WaitGroup
		wgResults.Add(1)
		go workerResults(log, cfg, &wgResults, chResult)

		log.Infof("Starting %d operations across %d threads...", len(cfg.Copiers), cfg.Threads)
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
		log.Verbosef("waiting for copy threads to complete")
		wgCopiers.Wait()
		close(chResult)
		log.Verbosef("waiting for results thread to complete")
		wgResults.Wait()

		log.Infof("Done")
		return nil
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatalf("%v", err)
	}
}

type Result struct {
	ThreadID int
	Err      error
	Msg      string
	Ctx      copier.Context
}

func workerCopy(
	log frog.Logger, cfg *Config, wg *sync.WaitGroup, threadID int,
	chCopier chan copier.Copier, chResult chan Result,
) {
	defer wg.Done()
	log.Verbosef("workerCopy %d: starting", threadID)
	count := 0
	for {
		c, ok := <-chCopier
		if !ok {
			log.Verbosef("workerCopy %d: closed after %d iterations", threadID, count)
			return
		}
		count++
		chResult <- Result{
			ThreadID: threadID,
			Err:      c.Copy(),
			Msg:      c.Dest(),
			Ctx:      c.Context(),
		}
	}
}

func workerResults(log frog.Logger, cfg *Config, wg *sync.WaitGroup, ch chan Result) {
	defer wg.Done()
	log.Verbosef("workerResults: starting")
	count := 0
	for {
		r, ok := <-ch
		if !ok {
			log.Verbosef("workerResults: closed after %d iterations", count)
			return
		}
		count++
		if r.Err != nil {
			log.Errorf(" ThreadID %d; Line %d: %v", r.ThreadID, r.Ctx.Line, r.Err)
		} else {
			log.Infof(" ThreadID %d; Line %d: %s", r.ThreadID, r.Ctx.Line, r.Msg)
		}
	}
}
