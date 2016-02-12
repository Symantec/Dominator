package filesystem

import (
	"github.com/Symantec/Dominator/lib/filter"
	"path"
)

func (fs *FileSystem) filter(filter *filter.Filter) *FileSystem {
	if filter == nil {
		return fs
	}
	if err := fs.RebuildInodePointers(); err != nil {
		panic(err)
	}
	newFS := new(FileSystem)
	newFS.InodeTable = make(InodeTable)
	newFS.DirectoryInode = *fs.DirectoryInode.filter(newFS, filter, "/")
	newFS.ComputeTotalDataBytes()
	return newFS
}

func (inode *DirectoryInode) filter(newFS *FileSystem,
	filter *filter.Filter, name string) *DirectoryInode {
	newInode := new(DirectoryInode)
	newInode.Mode = inode.Mode
	newInode.Uid = inode.Uid
	newInode.Gid = inode.Gid
	for _, entry := range inode.EntryList {
		subName := path.Join(name, entry.Name)
		if filter.Match(subName) {
			continue
		}
		var newEntry *DirectoryEntry
		if inode, ok := entry.inode.(*DirectoryInode); ok {
			newEntry = new(DirectoryEntry)
			newEntry.Name = entry.Name
			newEntry.InodeNumber = entry.InodeNumber
			newEntry.inode = inode.filter(newFS, filter, subName)
		} else {
			newEntry = entry
		}
		newInode.EntryList = append(newInode.EntryList, newEntry)
		newFS.InodeTable[entry.InodeNumber] = newEntry.inode
	}
	newFS.DirectoryCount++
	return newInode
}
