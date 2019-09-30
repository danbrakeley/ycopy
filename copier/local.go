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

func (f LocalFile) Context() Context {
	return f.ctx
}

func (f *LocalFile) SetContext(ctx Context) {
	f.ctx = ctx
}

func (f LocalFile) DebugPrint() string {
	return fmt.Sprintf("Copy from \"%s\" to \"%s\"", f.srcPath, f.destPath)
}

func (f LocalFile) Dest() string {
	return f.destPath
}

func (f LocalFile) Copy() error {
	if exists, isFolder, err := doesFileExist(f.srcPath); err != nil {
		return err
	} else if isFolder {
		return fmt.Errorf("is a folder, not a file: \"%s\"", f.srcPath)
	} else if !exists {
		return fmt.Errorf("does not exist: \"%s\"", f.srcPath)
	}

	if err := os.MkdirAll(filepath.Dir(f.destPath), os.ModePerm); err != nil {
		return err
	}

	if _, isFolder, err := doesFileExist(f.destPath); err != nil {
		return err
	} else if isFolder {
		return fmt.Errorf("path exists as a folder: \"%s\"", f.destPath)
	}

	return copyFileContents(f.srcPath, f.destPath)
}
