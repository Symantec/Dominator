package scanner

import (
	"fmt"
)

func compare(left *FileSystem, right *FileSystem, verbose bool) bool {
	if len(left.InodeTable) != len(right.InodeTable) {
		fmt.Printf("left vs. right: %d vs %d inodes\n",
			len(left.InodeTable), len(right.InodeTable))
		return false
	}
	return true
}
