package copier

import (
	"net/url"
	"path/filepath"
	"strings"
)

type Context struct {
	Line int
}

type Copier interface {
	Copy() (msg string, err error)
	SetContext(ctx Context)
	Context() Context
	DisplayIntent() string
}

func MakeCopierFromString(line, sourcePath, targetPath string) (Copier, error) {
	switch {
	case strings.HasPrefix(line, "http://") || strings.HasPrefix(line, "https://"):
		url, err := url.Parse(line)
		if err != nil {
			return nil, err
		}
		return &HTTPFile{
			url:      url,
			destPath: filepath.Join(targetPath, url.Path),
		}, nil
	default:
		return &LocalFile{
			srcPath:  filepath.Join(sourcePath, line),
			destPath: filepath.Join(targetPath, line),
		}, nil
	}
}
