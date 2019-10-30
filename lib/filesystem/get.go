package filesystem

import (
	"github.com/Cloud-Foundations/Dominator/lib/hash"
)

func (fs *FileSystem) getObjects() map[hash.Hash]uint64 {
	objects := make(map[hash.Hash]uint64)
	for _, inode := range fs.InodeTable {
		if inode, ok := inode.(*RegularInode); ok {
			if inode.Size > 0 {
				objects[inode.Hash] = inode.Size
			}
		}
	}
	return objects
}
