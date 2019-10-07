package copier

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"

	"github.com/pkg/errors"
)

type HTTPFile struct {
	url      *url.URL
	destPath string
	ctx      Context
	total    uint64
}

func (f *HTTPFile) Context() Context {
	return f.ctx
}

func (f *HTTPFile) SetContext(ctx Context) {
	f.ctx = ctx
}

func (f *HTTPFile) DebugPrint() string {
	return fmt.Sprintf("Download from \"%s\" to \"%s\"", f.url.String(), f.destPath)
}

func (f *HTTPFile) Dest() string {
	return f.destPath
}

func (f *HTTPFile) BytesWritten() uint64 {
	return f.total
}

func (f *HTTPFile) Copy(wp *WriteProgress) error {
	if err := os.MkdirAll(filepath.Dir(f.destPath), os.ModePerm); err != nil {
		return err
	}

	_, isFolder, err := doesFileExist(f.destPath)
	if err != nil {
		return err
	}
	if isFolder {
		return fmt.Errorf("path exists as a folder: \"%s\"", f.destPath)
	}

	// DOWNLOAD FILE
	err = downloadFile(f.destPath, f.url.String(), wp)
	f.total = wp.BytesWritten()
	return err
}

// https://golangcode.com/download-a-file-from-a-url/
// downloadFile will download a url to a local file. It's efficient because it will
// write as it downloads and not load the whole file into memory.
func downloadFile(filepath string, url string, wp *WriteProgress) error {
	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s", resp.Status)
	}

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	if wp != nil {
		// get length of file (for progress updates)
		str := resp.Header.Get("Content-Length")
		if len(str) > 0 {
			n, err := strconv.ParseUint(str, 10, 64)
			if err != nil {
				return errors.Wrap(err, "Content-Length header")
			}
			wp.SetGoal(n)
		}
		_, err = io.Copy(out, io.TeeReader(resp.Body, wp))
	} else {
		_, err = io.Copy(out, resp.Body)
	}

	return err
}
