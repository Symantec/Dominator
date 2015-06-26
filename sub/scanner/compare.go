package scanner

import (
	"fmt"
)

func compare(left *FileSystem, right *FileSystem, verbose bool) bool {
	if len(left.InodeTable) != len(right.InodeTable) {
		fmt.Printf("left vs. right: %d vs. %d inodes\n",
			len(left.InodeTable), len(right.InodeTable))
		return false
	}
	return compareDirectories(&left.Directory, &right.Directory, verbose)
}

func compareDirectories(left *Directory, right *Directory, verbose bool) bool {
	if left.name != right.name {
		fmt.Printf("left vs. right: %s vs. %s\n", left.name, right.name)
		return false
	}
	if len(left.FileList) != len(right.FileList) {
		fmt.Printf("left vs. right: %d vs. %d files\n",
			len(left.FileList), len(right.FileList))
		return false
	}
	if len(left.DirectoryList) != len(right.DirectoryList) {
		fmt.Printf("left vs. right: %d vs. %d subdirs\n",
			len(left.DirectoryList), len(right.DirectoryList))
		return false
	}
	return true
}
