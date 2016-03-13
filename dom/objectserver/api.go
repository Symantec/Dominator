package objectserver

import (
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/objectserver/filesystem"
	"io"
	"log"
)

type ObjectServer struct {
	objSrv *filesystem.ObjectServer
	logger *log.Logger
}

func NewObjectServer(objDir string, logger *log.Logger) (*ObjectServer, error) {
	return newObjectServer(objDir, logger)
}

func (objSrv *ObjectServer) GetObject(hashVal hash.Hash) (
	uint64, io.ReadCloser, error) {
	return objSrv.objSrv.GetObject(hashVal)
}
