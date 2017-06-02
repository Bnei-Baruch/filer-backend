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
	check(err)
	fl, err := fileindex.Load(bufio.NewReader(f), filter)
	check(err)
	f.Close()
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

func NewIndex(path string, pattern string) Index {
	return Index{List: make(IndexList, 0), SHA1: fileindex.NewFastSearch(), Path: path, Pattern: pattern}
}

func (idx *Index) Load() {
	indexes, err := GetIndexList(idx.Path, idx.Pattern)
	if err != nil {
		log.Println(err)
		return
	}

	list := make(IndexList, 0, 10)
	sha1 := fileindex.NewFastSearch()

	for _, f := range indexes {
		fullPath := idx.Path + "/" + f.Name()
		fl := load(fullPath)
		list = append(list, IndexFile{Path: fullPath, Mtime: f.ModTime().Unix(), Files: fl})
		sha1.AddList(fl)
	}

	idx.Lock()
	idx.List = list
	idx.SHA1 = sha1
	idx.Unlock()
}

func (idx *Index) IsModified() bool {
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
