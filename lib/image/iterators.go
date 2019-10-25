package image

import (
	"github.com/Cloud-Foundations/Dominator/lib/filesystem"
	"github.com/Cloud-Foundations/Dominator/lib/hash"
)

func (image *Image) forEachObject(objectFunc func(hash.Hash) error) error {
	for _, inode := range image.FileSystem.InodeTable {
		if inode, ok := inode.(*filesystem.RegularInode); ok {
			if inode.Size > 0 {
				if err := objectFunc(inode.Hash); err != nil {
					return err
				}
			}
		}
	}
	if image.ReleaseNotes != nil && image.ReleaseNotes.Object != nil {
		if err := objectFunc(*image.ReleaseNotes.Object); err != nil {
			return err
		}
	}
	if image.BuildLog != nil && image.BuildLog.Object != nil {
		if err := objectFunc(*image.BuildLog.Object); err != nil {
			return err
		}
	}
	return nil
}
