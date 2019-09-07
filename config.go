package main

import (
	"bufio"
	"os"
	"strings"

	"github.com/danbrakeley/ycopy/copier"
	"github.com/pkg/errors"
)

type Config struct {
	StopOnFirstError bool
	Copiers          []copier.Copier
}

func MakeCopiers(listFile, sourcePath, targetPath string) ([]copier.Copier, error) {
	f, err := os.Open(listFile)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to open %s", listFile)
	}
	defer f.Close()

	var copiers []copier.Copier

	scanner := bufio.NewScanner(f)
	curLine := 0
	for scanner.Scan() {
		curLine++
		line := scanner.Text()
		if strings.IndexRune(line, '#') == 0 {
			continue
		}
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		c, err := copier.MakeCopierFromString(line, sourcePath, targetPath)
		if err != nil {
			return nil, errors.Wrapf(err, "line %d", curLine)
		}
		c.SetContext(copier.Context{Line: curLine})
		copiers = append(copiers, c)
	}

	if err := scanner.Err(); err != nil {
		return nil, errors.Wrapf(err, "line %d", curLine)
	}

	return copiers, nil
}
