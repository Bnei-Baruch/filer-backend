package fileutils

import (
	"crypto/sha1"
	"encoding/hex"
	"io"
	"os"
)

const ioBufferSize = 64 * 1024

// Copy a file from the "src" to "dst". Calculate SHA1 for the source file.
//   - The "dst" must not exist.
//   - Remove the "dst" if error is triggered in the process.
func SHA1_Copy(src, dst string) ([]byte, int64, error) {
	var fdst *os.File

	fsrc, err := os.Open(src)
	if err != nil {
		return nil, 0, err
	}
	defer fsrc.Close()

	fdst, err = os.OpenFile(dst, os.O_WRONLY|os.O_EXCL|os.O_CREATE, 0600)
	if err == nil {
		defer func() {
			fdst.Close()
			if err == io.EOF {
				FileTouch(fsrc, dst)
			} else {
				os.Remove(dst)
			}
		}()
		buf := make([]byte, ioBufferSize)
		hasher := sha1.New()
		var n int64
		for {
			var nr int
			nr, err = fsrc.Read(buf)
			if nr == 0 {
				break
			}
			hasher.Write(buf[:nr])
			_, err = fdst.Write(buf[:nr])
			if err != nil {
				break
			}
			n += int64(nr)
		}
		if err == io.EOF {
			return hasher.Sum(nil), n, nil
		}
	}
	return nil, 0, err
}

// Calculate SHA1 of a file
func SHA1_File(path string) ([]byte, int64, os.FileInfo, error) {
	f, err := os.Open(path)
	if err == nil {
		defer f.Close()
		buf := make([]byte, ioBufferSize)
		hasher := sha1.New()
		var n int64
		n, err = io.CopyBuffer(hasher, f, buf)
		if err == nil {
			stat, _ := f.Stat()
			return hasher.Sum(nil), n, stat, nil
		}
	}
	return nil, 0, nil, err
}

// SHA1

func (h SHA1) Sha1Sum() []byte {
	return h.xx.Sum(nil)
}

func (h SHA1) Sha1Str() string {
	return hex.EncodeToString(h.xx.Sum(nil))
}

// Sha1Reader:
//   Calculate SHA1 for all reading through this Reader

func NewSha1Reader(r io.Reader) *Sha1Reader {
	return &Sha1Reader{
		Reader: r,
		SHA1:   SHA1{sha1.New()},
	}
}

func (r *Sha1Reader) Read(b []byte) (n int, err error) {
	n, err = r.Reader.Read(b)
	if err == nil {
		r.xx.Write(b[:n])
	}
	return n, err
}

// Sha1Writer:
//   Calculate SHA1 for all writing through this Writer

func NewSha1Writer(w io.Writer) *Sha1Writer {
	return &Sha1Writer{
		Writer: w,
		SHA1:   SHA1{sha1.New()},
	}
}

func (r *Sha1Writer) Write(b []byte) (n int, err error) {
	n, err = r.Writer.Write(b)
	if err == nil {
		r.xx.Write(b[:n])
	}
	return n, err
}
