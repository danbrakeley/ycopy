package copier

import (
	"net/url"
	"path/filepath"
	"strings"
)

type Context struct {
	Filename string
	Line     int
}

type Copier interface {
	Copy() (msg string, err error)
	SetContext(ctx Context)
	Context() Context
	DisplayIntent() string
}

func MakeCopierFromString(line, srcPath, destPath string) (Copier, error) {
	switch {
	case strings.HasPrefix(line, "http://") || strings.HasPrefix(line, "https://"):
		url, err := url.Parse(line)
		if err != nil {
			return nil, err
		}
		return &HTTPFile{
			url:      url,
			destPath: filepath.Join(destPath, url.Path),
		}, nil
	default:
		return &LocalFile{
			srcPath:  filepath.Join(srcPath, line),
			destPath: filepath.Join(destPath, line),
		}, nil
	}
}
