package image

import (
	"errors"
	"fmt"
	"path"

	"github.com/Cloud-Foundations/Dominator/lib/filesystem"
	"github.com/Cloud-Foundations/Dominator/lib/objectserver"
)

func (image *Image) verify() error {
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

func (image *Image) verifyObjects(checker objectserver.ObjectsChecker) error {
	missingObjects, err := image.ListMissingObjects(checker)
	if err != nil {
		return err
	}
	switch len(missingObjects) {
	case 0:
		return nil
	case 1:
		return fmt.Errorf("object: %x is not available", missingObjects[0])
	default:
		return fmt.Errorf("%d objects are not available", len(missingObjects))
	}
}

func (image *Image) verifyRequiredPaths(requiredPaths map[string]rune) error {
	if image.Filter == nil {
		return nil
	}
	fs := image.FileSystem
	filenameToInodeTable := fs.FilenameToInodeTable()
	for pathName, pathType := range requiredPaths {
		inum, ok := filenameToInodeTable[pathName]
		if !ok {
			return errors.New(
				"VerifyRequiredPaths(): missing path: " + pathName)
		}
		inode := fs.InodeTable[inum]
		switch pathType {
		case 'b', 'c', 'p':
			if _, ok := inode.(*filesystem.SpecialInode); !ok {
				return errors.New(
					"VerifyRequiredPaths(): path is not a special inode: " +
						pathName)
			}
		case 'd':
			if _, ok := inode.(*filesystem.DirectoryInode); !ok {
				return errors.New(
					"VerifyRequiredPaths(): path is not a directory: " +
						pathName)
			}
		case 'f':
			if _, ok := inode.(*filesystem.RegularInode); !ok {
				return errors.New(
					"VerifyRequiredPaths(): path is not a regular file: " +
						pathName)
			}
		case 'l':
			if _, ok := inode.(*filesystem.SymlinkInode); !ok {
				return errors.New(
					"VerifyRequiredPaths(): path is not a symlink: " +
						pathName)
			}
		}
	}
	return nil
}
