package filesystem

import (
	"github.com/Symantec/Dominator/lib/hash"
	"io"
)

type FileSystemObjectServer struct {
	topDirectory string
}

func NewObjectServer(topDirectory string) (*FileSystemObjectServer, error) {
	return &FileSystemObjectServer{topDirectory}, nil
}

func (objSrv *FileSystemObjectServer) CheckObject(hash hash.Hash) bool {
	return objSrv.checkObject(hash)
}

func (objSrv *FileSystemObjectServer) GetObjectReader(hash hash.Hash) (uint64,
	io.Reader, error) {
	return objSrv.getObjectReader(hash)
}

func (objSrv *FileSystemObjectServer) PutObject(size uint64, data []byte) (
	hash.Hash, error) {
	return objSrv.putObject(size, data)
}
