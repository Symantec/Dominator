package lib

import (
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/image"
	"log"
)

func (sub *Sub) buildMissingLists(image *image.Image, pushComputedFiles bool,
	logger *log.Logger) (
	[]hash.Hash, map[hash.Hash]struct{}) {
	objectsToFetch := make(map[hash.Hash]struct{})
	objectsToPush := make(map[hash.Hash]struct{})
	for inum, inode := range image.FileSystem.InodeTable {
		if rInode, ok := inode.(*filesystem.RegularInode); ok {
			if rInode.Size > 0 {
				objectsToFetch[rInode.Hash] = struct{}{}
			}
		} else if pushComputedFiles {
			if _, ok := inode.(*filesystem.ComputedRegularInode); ok {
				pathname := image.FileSystem.InodeToFilenamesTable()[inum][0]
				if inode, ok := sub.ComputedInodes[pathname]; !ok {
					logger.Printf(
						"fetchMissingObjects(%s): missing computed file: %s\n",
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
	fetchList := make([]hash.Hash, 0, len(objectsToFetch))
	for hashVal := range objectsToFetch {
		fetchList = append(fetchList, hashVal)
	}
	return fetchList, objectsToPush
}
