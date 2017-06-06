package fileindex

import "errors"

type (
	FileRec struct {
		Path  string
		Sha1  string
		Size  int64
		Mtime int64
	}

	FileList []FileRec
	ByPath   []FileRec
	BySize   []FileRec
	ByTime   []FileRec

	FilterFunc func(fr FileRec) bool

	FastSearch struct {
		sha1map map[string]FileList
		pathmap map[string]FileRec
	}
)

var (
	ErrLongLine = errors.New("Long line")
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
func (fr FileRec) Equal(or FileRec) bool {
	if fr.Size == or.Size && fr.Mtime == or.Mtime && fr.Sha1 == or.Sha1 && fr.Path == or.Path {
		return true
	}
	return false
}
