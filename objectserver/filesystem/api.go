package filesystem

import (
	"github.com/Symantec/Dominator/lib/hash"
	"io"
)

type FileSystemObjectServer struct {
	baseDir  string
	checkMap map[hash.Hash]bool // Only set if object is known.
}

func NewObjectServer(baseDir string) (*FileSystemObjectServer, error) {
	return newObjectServer(baseDir)
}

func (objSrv *FileSystemObjectServer) AddObject(data []byte,
	expectedHash *hash.Hash) (
	hash.Hash, error) {
	return objSrv.addObject(data, expectedHash)
}

func (objSrv *FileSystemObjectServer) CheckObject(hash hash.Hash) bool {
	return objSrv.checkObject(hash)
}

func (objSrv *FileSystemObjectServer) GetObjectReader(hash hash.Hash) (uint64,
	io.ReadCloser, error) {
	return objSrv.getObjectReader(hash)
}
