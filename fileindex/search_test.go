package fileindex

import (
	"bufio"
	"strings"
	"testing"
)

var (
	ll           []FileList
	totalrecords int
)

func init() {
	for _, x := range FilesRecords {
		totalrecords += x
	}

	ll = make([]FileList, 0, len(Files))
	for _, files := range Files {
		f := bufio.NewReader(strings.NewReader(files))
		if l, err := Load(f, nil); err == nil {
			ll = append(ll, l)
		}
	}
}

func newfs() *FastSearch {
	fs := NewFastSearch()
	for _, l := range ll {
		fs.AddList(l)
	}
	return fs
}

func check(t *testing.T, fs *FastSearch, pathexpected int, sha1expected int) {
	if pathexpected != len(fs.pathmap) {
		t.Errorf("'pathmap' records = %d, expected %d", len(fs.pathmap), pathexpected)
	}
	if sha1expected != len(fs.sha1map) {
		t.Errorf("'sha1map' records = %d, expected %d", len(fs.sha1map), sha1expected)
	}
}

func TestAddList(t *testing.T) {
	fs := newfs()
	check(t, fs, 0, totalrecords)

	l := ll[0][:1]
	l[0].Path = strings.Replace(l[0].Path, "/Files/", "/Duplicates/", 1)
	fs.AddList(l)
	check(t, fs, 0, totalrecords)
}

func TestRemove(t *testing.T) {
	fs := newfs()

	n := 5
	l := ll[0][:n]
	for _, fr := range l {
		fs.Remove(fr.Sha1)
	}
	check(t, fs, 0, totalrecords-n)
}

func TestRemovePath(t *testing.T) {
	t.Skip("test data does not contain path information for now")
	fs := newfs()

	n := 5
	l := ll[0][:n]
	for _, fr := range l {
		fs.RemovePath(fr.Path)
	}
	check(t, fs, 0, totalrecords-n)
}
