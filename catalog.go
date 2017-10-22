package main

import (
	"bufio"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"kbb1.com/fileindex"
	"kbb1.com/fileutils"
)

var (
	reDisk *regexp.Regexp
	reTape *regexp.Regexp
	reWIN  *regexp.Regexp

	storages sync.Map
)

func InitStorages() {
	reDisk, _ = regexp.Compile("^[0-9][0-9][0-9]$")
	reTape, _ = regexp.Compile("^(ltfs|lto)-[0-9-]*$")
	reWIN, _ = regexp.Compile("^[a-z]:$")
}

// list of index files
func GetIndexList(path string) IndexList {
	ft, err := fileutils.Collect(path)
	if err != nil {
		log.Println(err)
	}

	il := make([]IndexFile, 0, 10)
	for _, dir := range ft {
		for _, fi := range dir.List {
			if fi.Mode().IsRegular() && fi.Name()[0] != '.' {
				il = append(il, IndexFile{dir.FullPath(fi), fi.ModTime().Unix(), nil})
			}
		}
	}
	return il
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

// Type: IndexMain

func NewIndex(path string) *IndexMain {
	return &IndexMain{List: make(IndexList, 0), fs: fileindex.NewFastSearch(), Path: path}
}

func (idx *IndexMain) GetFS() (fs *fileindex.FastSearch) {
	p := unsafe.Pointer(&idx.fs)
	return (*fileindex.FastSearch)(atomic.LoadPointer((*unsafe.Pointer)(p)))
}

func (idx *IndexMain) IsModified() bool {
	indexes := GetIndexList(idx.Path)
	now := time.Now().Unix()

	idx.Lock()
	list := idx.List
	idx.Unlock()

	if len(indexes) != len(list) {
		return true
	}
	for _, idxfile := range indexes {
		// ignore files modified within last 5s
		if now-idxfile.Mtime < 5 {
			continue
		}

		il := list.FindPath(idxfile.Path)
		if il == nil || il.Mtime != idxfile.Mtime {
			return true
		}
	}
	return false
}

// Load all indexes recursively. Reload an index if modification time is changed.
func (idx *IndexMain) Load() {
	indexes := GetIndexList(idx.Path)

	idx.Lock()
	curlist := idx.List
	idx.Unlock()

	list := make(IndexList, 0, 10)
	fs := fileindex.NewFastSearch()
	for _, idxfile := range indexes {
		var fl fileindex.FileList

		curidx := curlist.FindPath(idxfile.Path)
		if curidx == nil || curidx.Mtime != idxfile.Mtime {
			fl = load(idxfile.Path)
			log.Printf("Loaded %d records from %s\n", len(fl), idxfile.Path)
		} else {
			fl = curidx.Files
		}
		list = append(list, IndexFile{Path: idxfile.Path, Mtime: idxfile.Mtime, Files: fl})
		fs.AddList(fl)
	}

	idx.Lock()
	idx.List = list
	idx.SetFS(fs)
	idx.Unlock()
}

func (idx *IndexMain) SetFS(fs *fileindex.FastSearch) {
	p := unsafe.Pointer(&idx.fs)
	atomic.StorePointer((*unsafe.Pointer)(p), unsafe.Pointer(fs))
}

// filter unnecessary files
func filter(fr *fileindex.FileRec, storage *fileindex.Storage) bool {
	path, name := filepath.Split(fr.Path)
	ext := filepath.Ext(name)
	if fr.Size == 0 || name == "Thumbs.db" || name == ".DS_Store" || ext == ".lnk" {
		return false
	}

	if storage != nil {
		if _, ok := storages.Load(storage.Id); !ok {
			storages.Store(storage.Id, storage)
		}
		fr.Device = storage
		return true
	}

	dirs := strings.Split(path, "/")
	if dirs[0] == "" {
		dirs = dirs[1:]
	}
	if len(dirs) < 2 {
		log.Println("Wrong path:", fr.Path)
		return false
	}

	id := "unknown"
	status := "offline"

	switch dirs[0] {
	case "mnt":
		if reDisk.Match([]byte(dirs[1])) {
			id = "disk-" + dirs[1]
			status = "nearline"
		} else {
			id = "f1-" + dirs[1]
			status = "online"
		}
	case "net":
		switch dirs[1] {
		case "nas":
			id = "nas-" + dirs[2]
			status = "online"
		case "server":
			switch dirs[2] {
			case "b", "original":
				id = "server-d:"
			case "r":
				id = "server-h:"
			case "buffer", "nas":
				id = "server-e:"
			}
			status = "online"
		}
	case "tape":
		if reTape.Match([]byte(dirs[1])) {
			id = dirs[1]
		}
	default:
		if reWIN.Match([]byte(dirs[0])) {
			id = "server-" + dirs[0]
			status = "online"
		}
	}

	if v, ok := storages.Load(id); ok {
		fr.Device = v.(*fileindex.Storage)
		return true
	}

	if id == "unknown" {
		log.Println("Unknown storage:", fr.Path, fr.Sha1)
	}

	storage = &fileindex.Storage{
		Id:       id,
		Status:   status,
		Access:   "local",
		Country:  "il",
		Location: "merkaz",
	}
	storages.Store(id, storage)
	fr.Device = storage

	return true
}

// import an index from path using filter
func load(path string) fileindex.FileList {
	f, err := os.Open(path)
	if err != nil {
		log.Println(err)
		return fileindex.FileList{}
	}
	defer f.Close()

	var storage *fileindex.Storage

	pp := strings.Split(path, "/")
	switch pp[len(pp)-2] {
	case "nl-nforce":
		storage = &fileindex.Storage{
			Id:       "0618278a-0602-4d7a-bb95-b1f176774490",
			Status:   "online",
			Access:   "internet",
			Country:  "nl",
			Location: "nforce",
		}
	case "ca-ovh":
		storage = &fileindex.Storage{
			Id:       "aa886ee5-5d9b-413a-baae-63079c89575d",
			Status:   "online",
			Access:   "internet",
			Country:  "ca",
			Location: "ovh",
		}
	case "ca-uri":
		storage = &fileindex.Storage{
			Id:       "fcae6eb0-6e24-436d-b01f-30ec1a0a4817",
			Status:   "online",
			Access:   "local",
			Country:  "ca",
			Location: "uri",
		}
	case "ru-piter":
		storage = &fileindex.Storage{
			Id:       "b569d59c-8b7f-41c6-b37a-1ceaeccc3a8a",
			Status:   "online",
			Access:   "local",
			Country:  "ru",
			Location: "piter",
		}
	}

	fl, err := fileindex.Load(bufio.NewReader(f), func(fr *fileindex.FileRec) bool {
		return filter(fr, storage)
	})
	if err != nil {
		log.Println(err)
	}
	return fl
}
