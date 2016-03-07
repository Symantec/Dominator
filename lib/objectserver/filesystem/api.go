package filesystem

import (
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/objectserver"
	"io"
	"log"
	"sync"
)

type ObjectServer struct {
	baseDir  string
	rwLock   sync.RWMutex         // Proect map mutations.
	sizesMap map[hash.Hash]uint64 // Only set if object is known.
	logger   *log.Logger
}

func NewObjectServer(baseDir string, logger *log.Logger) (
	*ObjectServer, error) {
	return newObjectServer(baseDir, logger)
}

func (objSrv *ObjectServer) AddObject(reader io.Reader, length uint64,
	expectedHash *hash.Hash) (hash.Hash, bool, error) {
	return objSrv.addObject(reader, length, expectedHash)
}

func (objSrv *ObjectServer) CheckObjects(hashes []hash.Hash) ([]uint64, error) {
	return objSrv.checkObjects(hashes)
}

func (objSrv *ObjectServer) GetObjects(hashes []hash.Hash) (
	objectserver.ObjectsReader, error) {
	return objSrv.getObjects(hashes)
}

func (objSrv *ObjectServer) GetObject(hashVal hash.Hash) (
	uint64, io.ReadCloser, error) {
	return objSrv.getObject(hashVal)
}

func (objSrv *ObjectServer) ListObjects() []hash.Hash {
	return objSrv.listObjects()
}

func (objSrv *ObjectServer) NumObjects() uint64 {
	objSrv.rwLock.RLock()
	defer objSrv.rwLock.RUnlock()
	return uint64(len(objSrv.sizesMap))
}

type ObjectsReader struct {
	objectServer *ObjectServer
	hashes       []hash.Hash
	nextIndex    int64
}

func (or *ObjectsReader) Close() error {
	return nil
}

func (or *ObjectsReader) NextObject() (uint64, io.ReadCloser, error) {
	return or.nextObject()
}
