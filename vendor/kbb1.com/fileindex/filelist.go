package fileindex

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"sort"
)

const newListCapacity = 10000

// Imports records from r. The input format is
//   ["Path", "Sha1", Size, Mtime]
// It returns error and empty FileList:
//   - if r contains a long line
//   - if r contains a wrong data line
// Records can be filtered with filter.The nil filter does nothing.
func Load(r *bufio.Reader, filter FilterFunc) (FileList, error) {
	var data []interface{}
	fl := make(FileList, 0, newListCapacity)
	for {
		line, isPrefix, err := r.ReadLine()
		if err != nil && err != io.EOF {
			return FileList{}, err
		}
		if isPrefix {
			return FileList{}, ErrLongLine
		}
		if len(line) > 0 && line[0] != '#' {
			err := json.Unmarshal(line, &data)
			if err != nil {
				return FileList{}, err
			}
			if len(data) != 4 {
				return FileList{}, errors.New("Wrong line:" + string(line))
			}
			var fr FileRec
			fr.Path = data[0].(string)
			fr.Sha1 = data[1].(string)
			fr.Size = int64(data[2].(float64))
			fr.Mtime = int64(data[3].(float64))
			if filter == nil || filter(fr) {
				fl = append(fl, fr)
			}
		}
		if err == io.EOF {
			break
		}
	}
	return fl, nil
}

// Filter records and create a new FileList
func (fl FileList) Filter(filter FilterFunc) FileList {
	nl := make(FileList, 0, newListCapacity)
	for _, fr := range fl {
		if filter(fr) {
			nl = append(nl, fr)
		}
	}
	return nl
}

// Export records to w
func (fl FileList) Save(w io.Writer) error {
	data := make([]interface{}, 4)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	for _, fr := range fl {
		data[0] = fr.Path
		data[1] = fr.Sha1
		data[2] = fr.Size
		data[3] = fr.Mtime
		err := enc.Encode(&data)
		if err != nil {
			return err
		}
	}
	return nil
}

// Summary of Size for all records
func (fl FileList) Size() int64 {
	var size int64
	for _, fr := range fl {
		size += fr.Size
	}
	return size
}

func (fl FileList) SortByPath() {
	sort.Sort(ByPath(fl))
}

func (fl FileList) SortBySize() {
	sort.Sort(BySize(fl))
}

func (fl FileList) SortByTime() {
	sort.Sort(ByTime(fl))
}

// Split records of two FileLists using filter
func (fl FileList) Split(filter FilterFunc) (FileList, FileList) {
	l1 := make(FileList, 0, newListCapacity)
	l2 := make(FileList, 0, newListCapacity)
	for _, fr := range fl {
		if filter(fr) {
			l1 = append(l1, fr)
		} else {
			l2 = append(l2, fr)
		}
	}
	return l1, l2
}

// Split sorted records of two FileLists using filter
func (fl FileList) SplitSorted(filter FilterFunc) (FileList, FileList) {
	for i, fr := range fl {
		if !filter(fr) {
			return fl[:i], fl[i:]
		}
	}
	return fl, FileList{}
}

// Compare that two FileLists are equal
func (fl FileList) Equal(ol FileList) bool {
	if len(fl) != len(ol) {
		return false
	}
	for i, fr := range fl {
		if !fr.Equal(ol[i]) {
			return false
		}
	}
	return true
}
