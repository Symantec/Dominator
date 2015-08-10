package filesystem

import (
	"github.com/Symantec/Dominator/lib/hash"
)

func (objSrv *FileSystemObjectServer) putObject(data []byte,
	expectedHash *hash.Hash) (
	hash.Hash, error) {
	var hash hash.Hash
	// TODO(rgooch): Implement.
	return hash, nil
}
