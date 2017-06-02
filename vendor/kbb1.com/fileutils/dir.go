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

func Readdir(path string) (fl FileList, err error) {
	var f *os.File
	f, err = os.Open(path)
	if err == nil {
		defer f.Close()
		fl, err = f.Readdir(-1)
		if err != nil {
			fl = []os.FileInfo{}
		}
	}
	return fl, err
}

func ReaddirMatch(path string, filter FileFilter) (fl []os.FileInfo, err error) {
	fl, err = Readdir(path)
	if err != nil {
		return fl, err
	}

	res := make([]os.FileInfo, 0, 10)
	for _, fi := range fl {
		if filter.Match(fi) {
			res = append(res, fi)
		}
	}
	return res, nil
}

//func Collect(path string, shift string) {
//	f, err := os.Open(path)
//	if err == nil {
//		defer f.Close()
//		var fi []os.FileInfo
//		fi, err = f.Readdir(-1)
//		if err == nil {
//			for _, v := range fi {
//				fmt.Println(shift + v.Name())
//				if v.IsDir() {
//					Collect(path+"/"+v.Name(), shift+"  ")
//				}
//			}
//		}
//	}
//}
