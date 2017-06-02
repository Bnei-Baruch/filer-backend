package fileutils

import "syscall"

func DiskAvailable(path string) int64 {
	var statfs syscall.Statfs_t

	err := syscall.Statfs(path, &statfs)
	if err != nil {
		return -1
	} else {
		return statfs.Bsize * int64(statfs.Bavail)
	}
}
