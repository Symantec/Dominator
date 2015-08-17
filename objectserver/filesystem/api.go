package filesystem

import (
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/objectserver"
	"io"
)

type ObjectServer struct {
	baseDir  string
	sizesMap map[hash.Hash]uint64 // Only set if object is known.
}

func NewObjectServer(baseDir string) (*ObjectServer, error) {
	return newObjectServer(baseDir)
}

func (objSrv *ObjectServer) AddObjects(datas [][]byte,
	expectedHashes []*hash.Hash) ([]hash.Hash, error) {
	return objSrv.addObjects(datas, expectedHashes)
}

func (objSrv *ObjectServer) CheckObjects(hashes []hash.Hash) ([]uint64, error) {
	return objSrv.checkObjects(hashes)
}

func (objSrv *ObjectServer) GetObjects(hashes []hash.Hash) (
	objectserver.ObjectsReader, error) {
	return objSrv.getObjects(hashes)
}

type ObjectsReader struct {
	objectServer *ObjectServer
	hashes       []hash.Hash
	nextIndex    int64
}

func (or *ObjectsReader) NextObject() (uint64, io.ReadCloser, error) {
	return or.nextObject()
}
