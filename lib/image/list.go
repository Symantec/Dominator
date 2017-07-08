package image

import (
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/hash"
)

func (image *Image) listObjects() []hash.Hash {
	hashes := make([]hash.Hash, 0, image.FileSystem.NumRegularInodes+2)
	for _, inode := range image.FileSystem.InodeTable {
		if inode, ok := inode.(*filesystem.RegularInode); ok {
			if inode.Size > 0 {
				hashes = append(hashes, inode.Hash)
			}
		}
	}
	if image.ReleaseNotes != nil && image.ReleaseNotes.Object != nil {
		hashes = append(hashes, *image.ReleaseNotes.Object)
	}
	if image.BuildLog != nil && image.BuildLog.Object != nil {
		hashes = append(hashes, *image.BuildLog.Object)
	}
	return hashes
}
