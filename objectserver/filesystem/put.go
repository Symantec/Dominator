package filesystem

import (
	"github.com/Symantec/Dominator/lib/hash"
)

func (objSrv *FileSystemObjectServer) putObject(size uint64, data []byte) (
	hash.Hash, error) {
	var hash hash.Hash
	return hash, nil
}
