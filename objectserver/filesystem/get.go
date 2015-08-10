package filesystem

import (
	"github.com/Symantec/Dominator/lib/hash"
	"io"
)

func (objSrv *FileSystemObjectServer) getObjectReader(hash hash.Hash) (uint64,
	io.Reader, error) {
	// TODO(rgooch): Implement.
	return 0, nil, nil
}
