package fileindex

func NewFastSearch() FastSearch {
	return FastSearch{ll: make(map[string]FileList)}
}

func (fs FastSearch) AddList(fl FileList) {
	for _, f := range fl {
		if x, ok := fs.ll[f.Sha1]; ok {
			fs.ll[f.Sha1] = append(x, f)
		} else {
			fs.ll[f.Sha1] = []FileRec{f}
		}
	}
}

func (fs FastSearch) Search(sha1 string) (fl FileList, ok bool) {
	fl, ok = fs.ll[sha1]
	return
}
