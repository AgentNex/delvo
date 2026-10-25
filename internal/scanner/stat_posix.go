//go:build !windows

package scanner

import (
	"os"
	"syscall"
)

func getPlatformStat(fi os.FileInfo, defaultAllocated int64) (dev, ino uint64, allocated int64) {
	allocated = defaultAllocated
	if stat, ok := fi.Sys().(*syscall.Stat_t); ok {
		dev = uint64(stat.Dev)
		ino = uint64(stat.Ino)
		if stat.Blocks > 0 {
			allocated = stat.Blocks * 512
		}
	}
	return dev, ino, allocated
}
