package fileindex

import (
	"errors"
)

type (
	FileRec struct {
		Path   string
		Sha1   string
		Size   int64
		Mtime  int64
		Device *Storage
	}

	FileList []*FileRec
	FileMap  map[string]*FileRec

	ByPath []*FileRec
	BySize []*FileRec
	ByTime []*FileRec

	AddFunc    func(fr *FileRec)
	FilterFunc func(fr *FileRec) bool

	FastSearch struct {
		sha1map map[string]FileList
		pathmap FileMap
	}

	Storage struct {
		Id       string `json:"id" form:"id"`
		Status   string `json:"status" form:"status"`
		Access   string `json:"access" form:"access"`
		Country  string `json:"country" form:"country"`
		Location string `json:"location" form:"location"`
	}
)

var (
	ErrLongLine     = errors.New("Long line")
	ErrFileModified = errors.New("The file have been modified")
)

func (a ByPath) Len() int           { return len(a) }
func (a ByPath) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByPath) Less(i, j int) bool { return a[i].Path < a[j].Path }

func (a BySize) Len() int           { return len(a) }
func (a BySize) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a BySize) Less(i, j int) bool { return a[i].Size < a[j].Size }

func (a ByTime) Len() int           { return len(a) }
func (a ByTime) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByTime) Less(i, j int) bool { return a[i].Mtime < a[j].Mtime }

// Equal comapres two records and returns True in case all fields are equal
func (fr *FileRec) Equal(or *FileRec) bool {
	if fr.Size == or.Size && fr.Mtime == or.Mtime && fr.Sha1 == or.Sha1 && fr.Path == or.Path {
		return true
	}
	return false
}
