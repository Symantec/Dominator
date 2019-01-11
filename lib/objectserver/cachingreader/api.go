package cachingreader

import (
	"io"
	"sync"
	"time"

	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/objectserver"
)

type objectType struct {
	hash               hash.Hash
	size               uint64
	newer              *objectType
	older              *objectType
	usageCount         uint
	downloadingChannel <-chan struct{} // Closed then set to nil when finished.
}

type downloadingObject struct {
	size uint64
}

type ObjectServer struct {
	baseDir             string
	flushTimer          *time.Timer
	logger              log.DebugLogger
	lruUpdateNotifier   chan<- struct{}
	maxCachedBytes      uint64
	objectServerAddress string
	rwLock              sync.RWMutex // Protect the following fields.
	cachedBytes         uint64       // Includes lruBytes.
	downloadingBytes    uint64       // Objects being downloaded and cached.
	lruBytes            uint64       // Cached but not in-use objects.
	newest              *objectType  // For unused objects only.
	objects             map[hash.Hash]*objectType
	oldest              *objectType // For unused objects only.
}

func NewObjectServer(baseDir string, maxCachedBytes uint64,
	objectServerAddress string, logger log.DebugLogger) (*ObjectServer, error) {
	return newObjectServer(baseDir, maxCachedBytes, objectServerAddress, logger)
}

func (objSrv *ObjectServer) GetObjects(hashes []hash.Hash) (
	objectserver.ObjectsReader, error) {
	return objSrv.getObjects(hashes)
}

func (objSrv *ObjectServer) WriteHtml(writer io.Writer) {
	objSrv.writeHtml(writer)
}
