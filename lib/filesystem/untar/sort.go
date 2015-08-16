package untar

import (
	"github.com/Symantec/Dominator/lib/filesystem"
	"sort"
)

type regularFileList []*filesystem.RegularFile
type symlinkList []*filesystem.Symlink
type fileList []*filesystem.File
type directoryList []*filesystem.Directory

func sortDirectory(directory *filesystem.Directory) {
	// Sort regular files.
	var regularFileList regularFileList
	regularFileList = directory.RegularFileList
	sort.Sort(regularFileList)
	directory.RegularFileList = regularFileList
	// Sort symlinks.
	var symlinkList symlinkList
	symlinkList = directory.SymlinkList
	sort.Sort(symlinkList)
	directory.SymlinkList = symlinkList
	// Sort files.
	var fileList fileList
	fileList = directory.FileList
	sort.Sort(fileList)
	directory.FileList = fileList
	// Sort directories.
	var directoryList directoryList
	directoryList = directory.DirectoryList
	sort.Sort(directoryList)
	directory.DirectoryList = directoryList
	// Recurse through directories.
	for _, subdir := range directory.DirectoryList {
		sortDirectory(subdir)
	}
}

func (list regularFileList) Len() int {
	return len(list)
}

func (list regularFileList) Less(left, right int) bool {
	if list[left].Name < list[right].Name {
		return true
	}
	return false
}

func (list regularFileList) Swap(left, right int) {
	tmp := list[left]
	list[left] = list[right]
	list[right] = tmp
}

func (list symlinkList) Len() int {
	return len(list)
}

func (list symlinkList) Less(left, right int) bool {
	if list[left].Name < list[right].Name {
		return true
	}
	return false
}

func (list symlinkList) Swap(left, right int) {
	tmp := list[left]
	list[left] = list[right]
	list[right] = tmp
}

func (list fileList) Len() int {
	return len(list)
}

func (list fileList) Less(left, right int) bool {
	if list[left].Name < list[right].Name {
		return true
	}
	return false
}

func (list fileList) Swap(left, right int) {
	tmp := list[left]
	list[left] = list[right]
	list[right] = tmp
}

func (list directoryList) Len() int {
	return len(list)
}

func (list directoryList) Less(left, right int) bool {
	if list[left].Name < list[right].Name {
		return true
	}
	return false
}

func (list directoryList) Swap(left, right int) {
	tmp := list[left]
	list[left] = list[right]
	list[right] = tmp
}
