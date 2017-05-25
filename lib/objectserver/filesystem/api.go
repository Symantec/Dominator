package filesystem

import (
	"flag"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/objectserver"
	"io"
	"sync"
	"time"
)

var (
	objectServerCleanupStartPercent = flag.Int(
		"objectServerCleanupStartPercent", 95, "")
	objectServerCleanupStopPercent = flag.Int("objectServerCleanupStopPercent",
		90, "")
)

type ObjectServer struct {
	baseDir               string
	gc                    objectserver.GarbageCollector
	logger                log.Logger
	rwLock                sync.RWMutex         // Protect the following fields.
	sizesMap              map[hash.Hash]uint64 // Only set if object is known.
	lastGarbageCollection time.Time
	lastMutationTime      time.Time
}

func NewObjectServer(baseDir string, logger log.Logger) (
	*ObjectServer, error) {
	return newObjectServer(baseDir, logger)
}

// AddObject will add an object. Object data are read from reader (length bytes
// are read). The object hash is computed and compared with expectedHash if not
// nil. The following are returned:
//   computed hash value
//   a boolean which is true if the object is new
//   an error or nil if no error.
func (objSrv *ObjectServer) AddObject(reader io.Reader, length uint64,
	expectedHash *hash.Hash) (hash.Hash, bool, error) {
	return objSrv.addObject(reader, length, expectedHash)
}

func (objSrv *ObjectServer) CheckObjects(hashes []hash.Hash) ([]uint64, error) {
	return objSrv.checkObjects(hashes)
}

// CommitObject will commit (add) a previously stashed object.
func (objSrv *ObjectServer) CommitObject(hashVal hash.Hash) error {
	return objSrv.commitObject(hashVal)
}

func (objSrv *ObjectServer) DeleteObject(hashVal hash.Hash) error {
	return objSrv.deleteObject(hashVal)
}

func (objSrv *ObjectServer) DeleteStashedObject(hashVal hash.Hash) error {
	return objSrv.deleteStashedObject(hashVal)
}

func (objSrv *ObjectServer) SetGarbageCollector(
	gc objectserver.GarbageCollector) {
	objSrv.gc = gc
}

func (objSrv *ObjectServer) GetObject(hashVal hash.Hash) (
	uint64, io.ReadCloser, error) {
	return objectserver.GetObject(objSrv, hashVal)
}

func (objSrv *ObjectServer) GetObjects(hashes []hash.Hash) (
	objectserver.ObjectsReader, error) {
	return objSrv.getObjects(hashes)
}

func (objSrv *ObjectServer) LastMutationTime() time.Time {
	objSrv.rwLock.RLock()
	defer objSrv.rwLock.RUnlock()
	return objSrv.lastMutationTime
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
	return uint64(len(objSrv.sizesMap))
}

// StashOrVerifyObject will stash an object if it is new or it will verify if it
// already exists. Object data are read from reader (length bytes are read). The
// object hash is computed and compared with expectedHash if not nil.
// The following are returned:
//   computed hash value
//   the object data if the object is new, otherwise nil
//   an error or nil if no error.
func (objSrv *ObjectServer) StashOrVerifyObject(reader io.Reader,
	length uint64, expectedHash *hash.Hash) (hash.Hash, []byte, error) {
	return objSrv.stashOrVerifyObject(reader, length, expectedHash)
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
