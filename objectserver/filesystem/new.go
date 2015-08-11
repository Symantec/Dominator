package filesystem

import (
	"github.com/Symantec/Dominator/lib/hash"
)

func newObjectServer(topDirectory string) (*FileSystemObjectServer, error) {
	var objSrv FileSystemObjectServer
	objSrv.topDirectory = topDirectory
	objSrv.checkMap = make(map[hash.Hash]bool)
	return &objSrv, nil
}
