package filesystem

func (fs *FileSystem) rebuildPointers() {
	fs.Directory.rebuildPointers(fs)
}

func (directory *Directory) rebuildPointers(fs *FileSystem) {
	for _, entry := range directory.RegularFileList {
		entry.rebuildPointers(fs)
	}
	for _, entry := range directory.SymlinkList {
		entry.rebuildPointers(fs)
	}
	for _, entry := range directory.FileList {
		entry.rebuildPointers(fs)
	}
	for _, entry := range directory.DirectoryList {
		entry.rebuildPointers(fs)
	}
}

func (file *RegularFile) rebuildPointers(fs *FileSystem) {
	file.inode = fs.RegularInodeTable[file.InodeNumber]
}

func (symlink *Symlink) rebuildPointers(fs *FileSystem) {
	symlink.inode = fs.SymlinkInodeTable[symlink.InodeNumber]
}

func (file *File) rebuildPointers(fs *FileSystem) {
	file.inode = fs.InodeTable[file.InodeNumber]
}

func (fs *FileSystem) computeTotalDataBytes() {
	fs.TotalDataBytes = 0
	for _, inode := range fs.RegularInodeTable {
		fs.TotalDataBytes += uint64(inode.Size)
	}
}
