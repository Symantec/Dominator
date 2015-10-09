package imageserver

import (
	"encoding/gob"
	"github.com/Symantec/Dominator/lib/filesystem"
)

func init() {
	gob.Register(&filesystem.RegularInode{})
	gob.Register(&filesystem.SymlinkInode{})
	gob.Register(&filesystem.Inode{})
	gob.Register(&filesystem.DirectoryInode{})
}
