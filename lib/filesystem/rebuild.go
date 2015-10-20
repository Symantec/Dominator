package filesystem

import (
	"path"
)

func (fs *FileSystem) rebuildInodePointers() {
	fs.DirectoryInode.rebuildInodePointers(fs)
}

func (inode *DirectoryInode) rebuildInodePointers(fs *FileSystem) {
	for _, dirent := range inode.EntryList {
		dirent.inode = fs.InodeTable[dirent.InodeNumber]
		if inode, ok := dirent.inode.(*DirectoryInode); ok {
			inode.rebuildInodePointers(fs)
		}
	}
}

func (fs *FileSystem) buildFilenamesTable() {
	fs.FilenamesTable = make(FilenamesTable)
	fs.DirectoryInode.buildFilenamesTable(fs, "")
}

func (inode *DirectoryInode) buildFilenamesTable(fs *FileSystem, name string) {
	for _, dirent := range inode.EntryList {
		name := path.Join(name, dirent.Name)
		fs.FilenamesTable[dirent.InodeNumber] = append(
			fs.FilenamesTable[dirent.InodeNumber], name)
		if inode, ok := dirent.inode.(*DirectoryInode); ok {
			inode.buildFilenamesTable(fs, name)
		}
	}
}

func (fs *FileSystem) buildHashToInodesTable() {
	fs.HashToInodesTable = make(HashToInodesTable)
	for inum, inode := range fs.InodeTable {
		if inode, ok := inode.(*RegularInode); ok && inode.Size > 0 {
			fs.HashToInodesTable[inode.Hash] = append(
				fs.HashToInodesTable[inode.Hash], inum)
		}
	}
}

func (fs *FileSystem) computeTotalDataBytes() {
	fs.NumRegularInodes = 0
	fs.TotalDataBytes = 0
	for _, inode := range fs.InodeTable {
		if inode, ok := inode.(*RegularInode); ok {
			fs.NumRegularInodes++
			fs.TotalDataBytes += uint64(inode.Size)
		}
	}
}

func (inode *DirectoryInode) buildEntryMap() {
	inode.EntriesByName = make(map[string]*DirectoryEntry)
	for _, dirent := range inode.EntryList {
		inode.EntriesByName[dirent.Name] = dirent
		if inode, ok := dirent.inode.(*DirectoryInode); ok {
			inode.buildEntryMap()
		}
	}
}
