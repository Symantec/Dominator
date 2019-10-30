package memory

import (
	"github.com/Cloud-Foundations/Dominator/lib/hash"
)

func newObjectServer() *ObjectServer {
	objSrv := new(ObjectServer)
	objSrv.objectMap = make(map[hash.Hash][]byte)
	return objSrv
}
