package fileutils

import (
	"hash"
	"io"
	"os"
	"regexp"
	"strings"
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

	LogCtx struct {
		Path   string
		Reopen int
		out    io.WriteCloser
	}

	LogWriter struct {
		ctx LogCtx
	}
)

type (
	FileList []os.FileInfo

	Directory struct {
		Path string
		List FileList
	}

	FileTree []Directory

	FilterFunc func(fi os.FileInfo) bool

	FileFilter interface {
		Match(fi os.FileInfo) bool
	}

	NullFilter struct{}

	RegexpFilter struct {
		re *regexp.Regexp
	}
)

func AddSlash(path string) string {
	if len(path) > 0 && path[len(path)-1] != '/' {
		return path + "/"
	}
	return path
}

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

func IntIndex(x int, defvalue int) int {
	if x < 0 {
		return defvalue
	}
	return x
}

func BaseHostName() string {
	h, e := os.Hostname()
	if e == nil {
		return h[0:IntIndex(strings.IndexRune(h, '.'), len(h))]
	}
	return "localhost"
}
