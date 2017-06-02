package fileutils

import (
	"hash"
	"io"
	"os"
	"regexp"
	"time"
)

type (
	SHA1 struct {
		xx hash.Hash
	}

	Sha1Reader struct {
		io.Reader
		SHA1
	}

	Sha1Writer struct {
		io.Writer
		SHA1
	}
)

type (
	FileList []os.FileInfo

	FilterFunc func(fi os.FileInfo) bool

	FileFilter interface {
		Match(fi os.FileInfo) bool
	}

	NullFilter struct{}

	RegexpFilter struct {
		re *regexp.Regexp
	}
)

func FileTouch(src *os.File, dst string) error {
	s, err := src.Stat()
	if err == nil {
		err = os.Chtimes(dst, time.Now(), s.ModTime())
		if err == nil {
			err = os.Chmod(dst, s.Mode())
		}
	}
	return err
}
