package imageserver

import (
	"encoding/gob"
	"github.com/Symantec/Dominator/lib/filesystem"
)

func init() {
	gob.Register(&filesystem.RegularInode{})
	gob.Register(&filesystem.ComputedRegularInode{})
	gob.Register(&filesystem.SymlinkInode{})
	gob.Register(&filesystem.SpecialInode{})
	gob.Register(&filesystem.DirectoryInode{})
}
