package copier

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

type HTTPFile struct {
	url      *url.URL
	destPath string
	ctx      Context
}

func (f HTTPFile) DisplayIntent() string {
	return fmt.Sprintf("Download from \"%s\" to \"%s\"", f.url.String(), f.destPath)
}
func (f *HTTPFile) SetContext(ctx Context) {
	f.ctx = ctx
}

func (f HTTPFile) Context() Context {
	return f.ctx
}

func (f HTTPFile) Copy() (msg string, err error) {
	if err = os.MkdirAll(filepath.Dir(f.destPath), os.ModePerm); err != nil {
		return
	}

	var isFolder bool
	_, isFolder, err = doesFileExist(f.destPath)
	if err != nil {
		return
	}
	if isFolder {
		err = fmt.Errorf("path exists as a folder: \"%s\"", f.destPath)
		return
	}

	// DOWNLOAD FILE
	err = downloadFile(f.destPath, f.url.String())
	if err != nil {
		return
	}

	msg = fmt.Sprintf("%s", f.destPath)
	return
}

// https://golangcode.com/download-a-file-from-a-url/
// downloadFile will download a url to a local file. It's efficient because it will
// write as it downloads and not load the whole file into memory.
func downloadFile(filepath string, url string) error {

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}
