package fileindex

func NewFastSearch() *FastSearch {
	return &FastSearch{
		sha1map: make(map[string]FileList),
		pathmap: make(map[string]*FileRec),
	}
}

func (fs *FastSearch) Add(fr *FileRec) {
	if x, ok := fs.sha1map[fr.Sha1]; ok {
		fs.sha1map[fr.Sha1] = append(x, fr)
	} else {
		fs.sha1map[fr.Sha1] = []*FileRec{fr}
	}
	fs.pathmap[fr.Path] = fr
}

func (fs *FastSearch) AddList(fl FileList) {
	for _, fr := range fl {
		fs.Add(fr)
	}
}

func (fs *FastSearch) GetAll() (fl []FileList) {
	fl = make([]FileList, 0, len(fs.sha1map))
	for _, v := range fs.sha1map {
		fl = append(fl, v)
	}
	return
}

func (fs *FastSearch) Duplicate() *FastSearch {
	fsdup := FastSearch{
		sha1map: make(map[string]FileList, len(fs.sha1map)),
		pathmap: make(map[string]*FileRec, len(fs.pathmap)),
	}
	for k, v := range fs.sha1map {
		fsdup.sha1map[k] = v
	}
	for k, v := range fs.pathmap {
		fsdup.pathmap[k] = v
	}
	return &fsdup
}

func (fs *FastSearch) Remove(sha1 string) {
	if fl, ok := fs.sha1map[sha1]; ok {
		for _, fr := range fl {
			_ = fr
			delete(fs.pathmap, fr.Path)
		}
		delete(fs.sha1map, sha1)
	}
}

func (fs *FastSearch) RemovePath(path string) {
	if fr, ok := fs.pathmap[path]; ok {
		if fl, ok := fs.sha1map[fr.Sha1]; ok {
			if len(fl) == 1 {
				if fl[0].Path == path {
					delete(fs.sha1map, fr.Sha1)
				}
			} else {
				x := 0
				for i := 0; i < len(fl); i++ {
					if fl[i].Path != path && x != i {
						fl[x] = fl[i]
						x++
					}
				}
				if x > 0 {
					fs.sha1map[fr.Sha1] = fl[:x]
				} else {
					delete(fs.sha1map, fr.Sha1)
				}
			}
		}
		delete(fs.pathmap, path)
	}
}

func (fs *FastSearch) Search(sha1 string) (fl FileList, ok bool) {
	fl, ok = fs.sha1map[sha1]
	return
}

func (fs *FastSearch) SearchPath(path string) (fr *FileRec, ok bool) {
	fr, ok = fs.pathmap[path]
	return
}

func (fs *FastSearch) Update(fr *FileRec) {
	if frFound, pathFound := fs.pathmap[fr.Path]; pathFound {
		if frFound.Sha1 == fr.Sha1 {
			fs.pathmap[fr.Path] = fr
			return
		} else {
			fs.RemovePath(frFound.Path)
		}
	}
	fs.Add(fr)
}
