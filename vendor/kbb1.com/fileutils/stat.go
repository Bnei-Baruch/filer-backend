package fileutils

import (
	"os"
	"syscall"
)

func DiskAvailable(path string) int64 {
	var statfs syscall.Statfs_t

	err := syscall.Statfs(path, &statfs)
	if err != nil {
		return -1
	} else {
		return statfs.Bsize * int64(statfs.Bavail)
	}
}

func FileSize(path string) int64 {
	stat, err := os.Lstat(path)
	if err == nil {
		return stat.Size()
	}
	return -1
}
