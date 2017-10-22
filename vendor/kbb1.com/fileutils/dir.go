package fileutils

import (
	"os"
	"regexp"
)

func (ff NullFilter) Match(fi os.FileInfo) bool {
	return false
}

func (ff RegexpFilter) Match(fi os.FileInfo) bool {
	return ff.re.MatchString(fi.Name())
}

func NewRegexpFilter(expr string) FileFilter {
	re, err := regexp.Compile(expr)
	if err != nil {
		return NullFilter{}
	}
	return RegexpFilter{re: re}
}

func Readdir(path string) (dir Directory, err error) {
	var f *os.File
	fl := []os.FileInfo{}
	f, err = os.Open(path)
	if err == nil {
		defer f.Close()
		fl, err = f.Readdir(-1)
		if err != nil {
			fl = []os.FileInfo{}
		}
	}
	return Directory{path, fl}, err
}

func ReaddirMatch(path string, filter FileFilter) (dir Directory, err error) {
	dir, err = Readdir(path)
	if err != nil {
		return dir, err
	}

	fl := make([]os.FileInfo, 0, 10)
	for _, fi := range dir.List {
		if filter.Match(fi) {
			fl = append(fl, fi)
		}
	}
	return Directory{path, fl}, nil
}

// Collect files recursively
func Collect(path string) (FileTree, error) {
	ft := make(FileTree, 0)
	dir, err := Readdir(path)
	if err != nil {
		return ft, err
	}

	fl := make([]os.FileInfo, 0, 10)
	for _, fi := range dir.List {
		if fi.IsDir() {
			f, e := Collect(dir.Path + "/" + fi.Name())
			if e == nil {
				ft = append(ft, f...)
			}
		}
		if fi.Mode().IsRegular() {
			fl = append(fl, fi)
		}
	}
	ft = append(ft, Directory{path, fl})

	return ft, nil
}

func (dir Directory) FullPath(fi os.FileInfo) string {
	return dir.Path + "/" + fi.Name()
}

func (ft FileTree) Files() []string {
	files := make([]string, 0, 10)
	for _, dir := range ft {
		for _, fi := range dir.List {
			if fi.Mode().IsRegular() {
				files = append(files, dir.FullPath(fi))
			}
		}
	}
	return files
}

type (
	Concurrency struct {
		current uint32
		limit   uint32
	}
)

func (c *Concurrency) Get() bool {
	if c.current >= c.limit {
		return false
	}
	c.limit++
	return true
}

func (c *Concurrency) Done() {
	if c.current > 0 {
		c.current--
	}
}

func (c *Concurrency) Wait() {

}

//func collect(path string, concurrency *int32) {
//	if f, err := os.Open(path); err != nil {
//		return err
//	}

//	files, err := f.Readdir(-1)
//	f.Close()
//	if err != nil {
//		return err
//	}

//	for _, x := range files {
//		if x.IsDir() {
//			p := path + "/" + x.Name()
//			z := atomic.AddInt64(&Concurency, -1)
//			if z > 0 {
//				go func() {
//					readdir(p)
//					atomic.AddInt64(&Concurency, 1)
//				}()
//			} else {
//				atomic.AddInt64(&Concurency, 1)
//				readdir(p)
//			}
//		}
//	}
//}
