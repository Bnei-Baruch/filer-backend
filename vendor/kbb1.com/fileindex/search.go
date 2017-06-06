package fileindex

func NewFastSearch() FastSearch {
	return FastSearch{
		sha1map: make(map[string]FileList),
		pathmap: make(map[string]FileRec),
	}
}

func (fs FastSearch) AddList(fl FileList) {
	for _, f := range fl {
		if x, ok := fs.sha1map[f.Sha1]; ok {
			fs.sha1map[f.Sha1] = append(x, f)
		} else {
			fs.sha1map[f.Sha1] = []FileRec{f}
		}
		fs.pathmap[f.Path] = f
	}
}

func (fs FastSearch) Remove(sha1 string) {
	if fl, ok := fs.sha1map[sha1]; ok {
		for _, fr := range fl {
			delete(fs.pathmap, fr.Path)
		}
		delete(fs.sha1map, sha1)
	}
}

func (fs FastSearch) RemovePath(path string) {
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

func (fs FastSearch) Search(sha1 string) (fl FileList, ok bool) {
	fl, ok = fs.sha1map[sha1]
	return
}

func (fs FastSearch) SearchPath(path string) (fr FileRec, ok bool) {
	fr, ok = fs.pathmap[path]
	return
}
