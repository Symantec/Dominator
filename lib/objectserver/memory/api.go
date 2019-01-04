package memory

import (
	"io"
	"sync"

	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/objectserver"
)

type ObjectServer struct {
	rwLock    sync.RWMutex // Protect map mutations.
	objectMap map[hash.Hash][]byte
}

func NewObjectServer() *ObjectServer {
	return newObjectServer()
}

func (objSrv *ObjectServer) AddObject(reader io.Reader, length uint64,
	expectedHash *hash.Hash) (hash.Hash, bool, error) {
	return objSrv.addObject(reader, length, expectedHash)
}

func (objSrv *ObjectServer) CheckObjects(hashes []hash.Hash) ([]uint64, error) {
	return objSrv.checkObjects(hashes)
}

func (objSrv *ObjectServer) GetObject(hashVal hash.Hash) (
	uint64, io.ReadCloser, error) {
	return objSrv.getObject(hashVal)
}

func (objSrv *ObjectServer) GetObjects(hashes []hash.Hash) (
	objectserver.ObjectsReader, error) {
	return objSrv.getObjects(hashes)
}

func (objSrv *ObjectServer) ListObjectSizes() map[hash.Hash]uint64 {
	return objSrv.listObjectSizes()
}

func (objSrv *ObjectServer) ListObjects() []hash.Hash {
	return objSrv.listObjects()
}

func (objSrv *ObjectServer) NumObjects() uint64 {
	objSrv.rwLock.RLock()
	defer objSrv.rwLock.RUnlock()
	return uint64(len(objSrv.objectMap))
}

type ObjectsReader struct {
	objectServer *ObjectServer
	hashes       []hash.Hash
	nextIndex    int64
	sizes        []uint64
}

func (or *ObjectsReader) Close() error {
	return nil
}

func (or *ObjectsReader) NextObject() (uint64, io.ReadCloser, error) {
	return or.nextObject()
}

func (or *ObjectsReader) ObjectSizes() []uint64 {
	return or.sizes
}
