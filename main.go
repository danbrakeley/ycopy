package main

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/danbrakeley/frog"
	"github.com/danbrakeley/ycopy/copier"
	"github.com/dustin/go-humanize"
	"github.com/urfave/cli"
)

const (
	VersionMaj   = 0
	VersionMin   = 3
	VersionPatch = 0
)

func main() {
	status := mainExit()
	if status != 0 {
		// From os/proc.go: "For portability, the status code should be in the range [0, 125]."
		if status < 0 || status > 125 {
			status = 125
		}
		os.Exit(status)
	}
}

func mainExit() int {
	log := frog.New(frog.Auto)
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
			log.Error("Unexpected arguments (options go before <list-file>, see --help)", frog.String("args", strings.Join(args[1:], " ")))
			cli.ShowAppHelpAndExit(c, 1)
		}
		list := strings.TrimSpace(args.Get(0))
		if len(list) == 0 {
			log.Error("Argument <list-file> is missing, but is required.")
			cli.ShowAppHelpAndExit(c, 1)
		}

		src, err := filepath.Abs(c.String("src"))
		if err != nil {
			log.Error("problem with --src", frog.Err(err))
			cli.ShowAppHelpAndExit(c, 1)
		}

		dest, err := filepath.Abs(c.String("dest"))
		if err != nil {
			log.Error("problem with --dest", frog.Err(err))
			cli.ShowAppHelpAndExit(c, 1)
		}

		copiers, err := MakeCopiers(list, src, dest)
		if err != nil {
			log.Error("problem with list-file", frog.Err(err), frog.String("list_file", list), frog.String("src", src), frog.String("dest", dest))
			cli.ShowAppHelpAndExit(c, 1)
		}

		cfg := &Config{
			Threads: c.Int("threads"),
			DryRun:  c.Bool("dryrun"),
			Copiers: copiers,
		}

		if cfg.DryRun {
			log.Info("dry run", frog.Int("num_threads", cfg.Threads), frog.Int("num_ops", len(cfg.Copiers)))
			for i, c := range cfg.Copiers {
				log.Info("", frog.Int("num", i), frog.String("op", c.DebugPrint()))
			}
			return nil
		}

		log.Info("Starting...", frog.Int("num_threads", cfg.Threads), frog.Int("num_ops", len(cfg.Copiers)))

		// install signal hanlder
		chSignal := make(chan os.Signal, 1)
		signal.Notify(chSignal, os.Interrupt)
		signalHandler := func(s os.Signal) {
			log.Verbose("got signal", frog.String("signal", s.String()))
			log.Warning("Interrupt detected, running actions will complete, but new actions will not be started...")
		}

		chCopier := make(chan copier.Copier)

		var wg sync.WaitGroup
		wg.Add(cfg.Threads)
		for i := 0; i < cfg.Threads; i++ {
			threadID := i + 1
			ll := frog.AddAnchor(log)
			go func() {
				workerCopy(ll, cfg, threadID, chCopier)
				frog.RemoveAnchor(ll)
				wg.Done()
			}()
		}

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
		log.Verbose("waiting for copy threads to complete")
		wg.Wait()

		log.Info("Done")
		return nil
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Error("aborting due to error", frog.Err(err))
		return 1
	}

	return 0
}

func workerCopy(log frog.Logger, cfg *Config, threadID int, chCopier chan copier.Copier) {
	log.Verbose("worker thread starting", frog.Int("thread", threadID))
	count := 0
	for {
		c, ok := <-chCopier
		if !ok {
			break
		}
		count++
		log.Transient(fmt.Sprintf(" + [%d] Copying %s...", threadID, c.Dest()))

		wl := copier.NewWriteProgress(
			time.Duration(100)*time.Millisecond,
			func(n, goal uint64) {
				log.Transient(fmt.Sprintf(" + [%d] %s: %s / %s (%.1f%%)", threadID, c.Dest(),
					humanize.Bytes(n), humanize.Bytes(goal), 100.0*float64(n)/float64(goal)))
			},
		)
		err := c.Copy(&wl)
		if err != nil {
			ctx := c.Context()
			log.Error("error during copy",
				frog.Int("thread", threadID),
				frog.Err(err),
				frog.String("file", ctx.Filename),
				frog.Int("line", ctx.Line),
			)
			continue
		}

		log.Info("File complete",
			frog.Int("thread", threadID),
			frog.String("dest", c.Dest()),
			frog.Uint64("bytes", c.BytesWritten()),
			frog.String("bytes_human", humanize.Bytes(c.BytesWritten())),
		)
	}
	log.Verbose("worker thread closing", frog.Int("thread", threadID), frog.Int("iterations", count))
}
