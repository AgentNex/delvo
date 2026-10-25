//go:build windows

package scanner

import (
	"os"
)

func getPlatformStat(fi os.FileInfo, defaultAllocated int64) (dev, ino uint64, allocated int64) {
	return 0, 0, defaultAllocated
}
