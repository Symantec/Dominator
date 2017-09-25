package untar

import (
	"sort"

	"github.com/Symantec/Dominator/lib/filesystem"
)

type directoryEntryList []*filesystem.DirectoryEntry

func sortDirectory(directory *filesystem.DirectoryInode) {
	var entryList directoryEntryList
	entryList = directory.EntryList
	sort.Sort(entryList)
	directory.EntryList = entryList
	// Recurse through directories.
	for _, dirent := range directory.EntryList {
		if inode, ok := dirent.Inode().(*filesystem.DirectoryInode); ok {
			sortDirectory(inode)
		}
	}
}

func (list directoryEntryList) Len() int {
	return len(list)
}

func (list directoryEntryList) Less(left, right int) bool {
	if list[left].Name < list[right].Name {
		return true
	}
	return false
}

func (list directoryEntryList) Swap(left, right int) {
	list[left], list[right] = list[right], list[left]
}
