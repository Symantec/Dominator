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
		fs.addFilenameToTable(dirent.InodeNumber, name)
		if inode, ok := dirent.inode.(*DirectoryInode); ok {
			inode.buildFilenamesTable(fs, name)
		}
	}
}

func (fs *FileSystem) addFilenameToTable(inode uint64, name string) {
	filenames := fs.FilenamesTable[inode]
	filenames = append(filenames, name)
	fs.FilenamesTable[inode] = filenames
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
