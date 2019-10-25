package lib

import (
	"github.com/Cloud-Foundations/Dominator/lib/filesystem"
	"github.com/Cloud-Foundations/Dominator/lib/hash"
	"github.com/Cloud-Foundations/Dominator/lib/image"
	"github.com/Cloud-Foundations/Dominator/lib/log"
)

func (sub *Sub) buildMissingLists(image *image.Image, pushComputedFiles bool,
	ignoreMissingComputedFiles bool, logger log.Logger) (
	map[hash.Hash]uint64, map[hash.Hash]struct{}) {
	objectsToFetch := make(map[hash.Hash]uint64)
	objectsToPush := make(map[hash.Hash]struct{})
	for inum, inode := range image.FileSystem.InodeTable {
		if rInode, ok := inode.(*filesystem.RegularInode); ok {
			if rInode.Size > 0 {
				objectsToFetch[rInode.Hash] = rInode.Size
			}
		} else if pushComputedFiles {
			if _, ok := inode.(*filesystem.ComputedRegularInode); ok {
				pathname := image.FileSystem.InodeToFilenamesTable()[inum][0]
				if inode, ok := sub.ComputedInodes[pathname]; !ok {
					if ignoreMissingComputedFiles {
						continue
					}
					logger.Printf(
						"buildMissingLists(%s): missing computed file: %s\n",
						sub, pathname)
					return nil, nil
				} else {
					objectsToPush[inode.Hash] = struct{}{}
				}
			}
		}
	}
	for _, hashVal := range sub.ObjectCache {
		delete(objectsToFetch, hashVal)
		delete(objectsToPush, hashVal)
	}
	for _, inode := range sub.FileSystem.InodeTable {
		if inode, ok := inode.(*filesystem.RegularInode); ok {
			if inode.Size > 0 {
				delete(objectsToFetch, inode.Hash)
				delete(objectsToPush, inode.Hash)
			}
		}
	}
	return objectsToFetch, objectsToPush
}
