package fileutils

func (fl FileList) Filter(filter FilterFunc) FileList {
	nl := make(FileList, 0, 1000)
	for _, fr := range fl {
		if filter(fr) {
			nl = append(nl, fr)
		}
	}
	return nl
}

func (fl FileList) Split(filter FilterFunc) (FileList, FileList) {
	l1 := make(FileList, 0, 1000)
	l2 := make(FileList, 0, 1000)
	for _, fr := range fl {
		if filter(fr) {
			l1 = append(l1, fr)
		} else {
			l2 = append(l2, fr)
		}
	}
	return l1, l2
}
