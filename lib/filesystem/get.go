package filesystem

import (
	"github.com/Symantec/Dominator/lib/hash"
)

func (fs *FileSystem) getObjects() map[hash.Hash]uint64 {
	objects := make(map[hash.Hash]uint64)
	for _, inode := range fs.InodeTable {
		if inode, ok := inode.(*RegularInode); ok {
			objects[inode.Hash] = inode.Size
		}
	}
	return objects
}
