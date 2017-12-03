package filesystem

import (
	"errors"
	"fmt"
	"path"
)

func (fs *FileSystem) rebuildInodePointers() error {
	return fs.DirectoryInode.rebuildInodePointers(fs)
}

func (inode *DirectoryInode) rebuildInodePointers(fs *FileSystem) error {
	for _, dirent := range inode.EntryList {
		tableInode, ok := fs.InodeTable[dirent.InodeNumber]
		if !ok {
			return fmt.Errorf("%s: no entry in inode table for: %d %p",
				dirent.Name, dirent.InodeNumber, dirent.inode)
		}
		if tableInode == nil {
			return fmt.Errorf("%s: nil entry in inode table for: %d %p",
				dirent.Name, dirent.InodeNumber, dirent.inode)
		} else if dirent.inode != nil && dirent.inode != tableInode {
			return fmt.Errorf(
				"%s: changing inode entry for: %d from: %p to %p\n",
				dirent.Name, dirent.InodeNumber, dirent.inode, tableInode)
		}
		dirent.inode = tableInode
		if inode, ok := dirent.inode.(*DirectoryInode); ok {
			if err := inode.rebuildInodePointers(fs); err != nil {
				return errors.New(dirent.Name + "/" + err.Error())
			}
		}
	}
	return nil
}

func (fs *FileSystem) buildInodeToFilenamesTable() InodeToFilenamesTable {
	if fs.inodeToFilenamesTable == nil {
		fs.inodeToFilenamesTable = make(InodeToFilenamesTable)
		fs.DirectoryInode.buildInodeToFilenamesTable(fs, "/")
	}
	return fs.inodeToFilenamesTable
}

func (inode *DirectoryInode) buildInodeToFilenamesTable(fs *FileSystem,
	name string) {
	for _, dirent := range inode.EntryList {
		name := path.Join(name, dirent.Name)
		fs.inodeToFilenamesTable[dirent.InodeNumber] = append(
			fs.inodeToFilenamesTable[dirent.InodeNumber], name)
		if inode, ok := dirent.inode.(*DirectoryInode); ok {
			inode.buildInodeToFilenamesTable(fs, name)
		}
	}
}

func (fs *FileSystem) buildFilenameToInodeTable() FilenameToInodeTable {
	if fs.filenameToInodeTable == nil {
		fs.filenameToInodeTable = make(map[string]uint64)
		for inum, filenames := range fs.InodeToFilenamesTable() {
			for _, filename := range filenames {
				fs.filenameToInodeTable[filename] = inum
			}
		}

	}
	return fs.filenameToInodeTable
}

func (fs *FileSystem) buildHashToInodesTable() HashToInodesTable {
	if fs.hashToInodesTable == nil {
		fs.hashToInodesTable = make(HashToInodesTable)
		for inum, inode := range fs.InodeTable {
			if inode, ok := inode.(*RegularInode); ok && inode.Size > 0 {
				fs.hashToInodesTable[inode.Hash] = append(
					fs.hashToInodesTable[inode.Hash], inum)
			}
		}
	}
	return fs.hashToInodesTable
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

func (fs *FileSystem) computeNumComputedRegularInodes() uint64 {
	if fs.numComputedRegularInodes == nil {
		var numInodes uint64
		for _, inode := range fs.InodeTable {
			if _, ok := inode.(*ComputedRegularInode); ok {
				numInodes++
			}
		}
		fs.numComputedRegularInodes = &numInodes
	}
	return *fs.numComputedRegularInodes
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

func (inode *DirectoryInode) replaceStrings(replaceFunc func(string) string) {
	for _, dirent := range inode.EntryList {
		dirent.Name = replaceFunc(dirent.Name)
		if inode, ok := dirent.inode.(*DirectoryInode); ok {
			inode.replaceStrings(replaceFunc)
		}
	}
}
