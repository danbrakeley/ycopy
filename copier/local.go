package copier

import (
	"fmt"
	"os"
	"path/filepath"
)

type LocalFile struct {
	srcPath  string
	destPath string
	ctx      Context
}

func (f LocalFile) DisplayIntent() string {
	return fmt.Sprintf("Copy from \"%s\" to \"%s\"", f.srcPath, f.destPath)
}

func (f *LocalFile) SetContext(ctx Context) {
	f.ctx = ctx
}

func (f LocalFile) Context() Context {
	return f.ctx
}

func (f LocalFile) Copy() (msg string, err error) {
	var exists, isFolder bool
	exists, isFolder, err = doesFileExist(f.srcPath)
	if err != nil {
		return
	}
	if isFolder {
		err = fmt.Errorf("is a folder, not a file: \"%s\"", f.srcPath)
		return
	}
	if !exists {
		err = fmt.Errorf("does not exist: \"%s\"", f.srcPath)
		return
	}

	if err = os.MkdirAll(filepath.Dir(f.destPath), os.ModePerm); err != nil {
		return
	}

	_, isFolder, err = doesFileExist(f.destPath)
	if err != nil {
		return
	}
	if isFolder {
		err = fmt.Errorf("path exists as a folder: \"%s\"", f.destPath)
		return
	}

	err = copyFileContents(f.srcPath, f.destPath)
	if err != nil {
		return
	}

	msg = fmt.Sprintf("%s", f.destPath)
	return
}
