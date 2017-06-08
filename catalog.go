package main

import (
	"bufio"
	"log"
	"os"
	"path/filepath"
	"regexp"

	"kbb1.com/fileindex"
	"kbb1.com/fileutils"
)

type (
	MyFileFilter struct {
		re *regexp.Regexp
	}
)

func (ff MyFileFilter) Match(fi os.FileInfo) bool {
	return fi.Mode().IsRegular() && ff.re.MatchString(fi.Name())
}

func NewFileFilter(expr string) fileutils.FileFilter {
	re, err := regexp.Compile(expr)
	if err != nil {
		return fileutils.NullFilter{}
	}
	return MyFileFilter{re: re}
}

// list of files matched the pattern
func GetIndexList(path string, pattern string) (fileutils.FileList, error) {
	return fileutils.ReaddirMatch(path, NewFileFilter(pattern))
}

// filter unnecessary files
func filter(fr fileindex.FileRec) bool {
	_, name := filepath.Split(fr.Path)
	ext := filepath.Ext(name)
	if fr.Size > 0 && name != "Thumbs.db" && name != ".DS_Store" && ext != ".lnk" {
		return true
	}
	return false
}

// import an index from path using filter
func load(path string) fileindex.FileList {
	f, err := os.Open(path)
	if err != nil {
		log.Println(err)
		return fileindex.FileList{}
	}
	defer f.Close()

	fl, err := fileindex.Load(bufio.NewReader(f), filter)
	if err != nil {
		log.Println(err)
	}
	return fl
}

// Type: IndexList

func (il IndexList) FindPath(path string) *IndexFile {
	for _, i := range il {
		if i.Path == path {
			return &i
		}
	}
	return nil
}

// Type: Index

func NewIndex(path string, pattern string) *IndexMain {
	return &IndexMain{List: make(IndexList, 0), FS: fileindex.NewFastSearch(), Path: path, Pattern: pattern}
}

func (idx *IndexMain) Load() {
	indexes, err := GetIndexList(idx.Path, idx.Pattern)
	if err != nil {
		log.Println(err)
		return
	}

	idx.Lock()
	curlist := idx.List
	idx.Unlock()

	list := make(IndexList, 0, 10)
	fs := fileindex.NewFastSearch()
	for _, f := range indexes {
		var fl fileindex.FileList
		fullPath := idx.Path + "/" + f.Name()
		mtime := f.ModTime().Unix()

		curidx := curlist.FindPath(fullPath)
		if curidx == nil || curidx.Mtime != mtime {
			fl = load(fullPath)
			log.Printf("Loaded %d records from %s\n", len(fl), fullPath)
		} else {
			fl = curidx.Files
		}
		list = append(list, IndexFile{Path: fullPath, Mtime: mtime, Files: fl})
		fs.AddList(fl)
	}

	idx.Lock()
	idx.List = list
	idx.FS = fs
	idx.Unlock()
}

func (idx *IndexMain) IsModified() bool {
	indexes, err := GetIndexList(idx.Path, idx.Pattern)
	if err != nil {
		log.Println(err)
		return false
	}

	idx.Lock()
	list := idx.List
	idx.Unlock()

	if len(indexes) != len(list) {
		return true
	}
	for _, f := range indexes {
		path := idx.Path + "/" + f.Name()
		il := list.FindPath(path)
		if il == nil || il.Mtime != f.ModTime().Unix() {
			return true
		}
	}
	return false
}
