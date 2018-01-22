package filesystem

import (
	"path"
)

func (directory *DirectoryInode) forEachEntry(directoryName string,
	fn func(name string, inodeNumber uint64, inode GenericInode) error) error {
	for _, dirent := range directory.EntryList {
		name := path.Join(directoryName, dirent.Name)
		if err := fn(name, dirent.InodeNumber, dirent.inode); err != nil {
			return err
		}
		if inode, ok := dirent.inode.(*DirectoryInode); ok {
			if err := inode.forEachEntry(name, fn); err != nil {
				return err
			}
		}
	}
	return nil
}

func (fs *FileSystem) forEachFile(
	fn func(name string, inodeNumber uint64, inode GenericInode) error) error {
	if err := fn("/", 0, &fs.DirectoryInode); err != nil {
		return err
	}
	return fs.DirectoryInode.forEachEntry("/", fn)
}
