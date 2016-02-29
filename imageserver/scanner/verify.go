package scanner

import (
	"errors"
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/image"
	"path"
)

func verifyImage(image *image.Image) error {
	computedInodes := make(map[uint64]struct{})
	return verifyDirectory(&image.FileSystem.DirectoryInode, computedInodes, "")
}

func verifyDirectory(directoryInode *filesystem.DirectoryInode,
	computedInodes map[uint64]struct{}, name string) error {
	for _, dirent := range directoryInode.EntryList {
		if _, ok := dirent.Inode().(*filesystem.ComputedRegularInode); ok {
			if _, ok := computedInodes[dirent.InodeNumber]; ok {
				return errors.New("duplicate computed inode: " +
					path.Join(name, dirent.Name))
			}
			computedInodes[dirent.InodeNumber] = struct{}{}
		} else if inode, ok := dirent.Inode().(*filesystem.DirectoryInode); ok {
			verifyDirectory(inode, computedInodes, path.Join(name, dirent.Name))
		}
	}
	return nil
}
