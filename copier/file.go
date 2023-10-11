package copier

import (
	"io"
	"io/fs"
	"os"
)

func doesFileExist(path string) (exists, isFolder bool, err error) {
	var stat os.FileInfo
	stat, err = os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			err = nil
		}
		return
	}

	if stat.IsDir() {
		isFolder = true
		return
	}

	exists = true
	return
}

// copyFileContents copies the contents of the file named src to the file named by dst. The file
// will be created if it does not already exist. If the destination file exists, all it's contents
// will be replaced by the contents of the source file.
// Inspired by: https://stackoverflow.com/questions/21060945/simple-way-to-copy-a-file-in-golang
func copyFileContents(src, dst string, wp *WriteProgress) error {
	var err error
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()
	if wp != nil {
		var fi fs.FileInfo
		fi, err = in.Stat()
		if err != nil {
			return err
		}
		wp.SetGoal(uint64(fi.Size()))
		_, err = io.Copy(out, io.TeeReader(in, wp))
	} else {
		_, err = io.Copy(out, in)
	}
	if err != nil {
		return err
	}
	return out.Sync()
}
