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

func (objSrv *FileSystemObjectServer) AddObjects(datas [][]byte,
	expectedHashes []*hash.Hash) ([]hash.Hash, error) {
	return objSrv.addObjects(datas, expectedHashes)
}

func (objSrv *FileSystemObjectServer) CheckObjects(hashes []hash.Hash) (
	[]bool, error) {
	return objSrv.checkObjects(hashes)
}

func (objSrv *FileSystemObjectServer) GetObjectReader(hash hash.Hash) (
	uint64, io.ReadCloser, error) {
	return objSrv.getObjectReader(hash)
}
