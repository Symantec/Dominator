package scanner

func (fs *FileSystem) rebuildPointers() {
	fs.Directory.rebuildPointers(fs)
}

func (directory *Directory) rebuildPointers(fs *FileSystem) {
	directory.inode = fs.InodeTable[directory.InodeNumber]
	for _, file := range directory.FileList {
		file.rebuildPointers(fs)
	}
	for _, dir := range directory.DirectoryList {
		dir.rebuildPointers(fs)
	}
}

func (file *File) rebuildPointers(fs *FileSystem) {
	file.inode = fs.InodeTable[file.InodeNumber]
}
