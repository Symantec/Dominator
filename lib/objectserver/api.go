package objectserver

import (
	"github.com/Symantec/Dominator/lib/hash"
	"io"
	"time"
)

type FullObjectServer interface {
	DeleteObject(hashVal hash.Hash) error
	ObjectServer
	LastMutationTime() time.Time
	ListObjectSizes() map[hash.Hash]uint64
	ListObjects() []hash.Hash
	NumObjects() uint64
}

type AddCallback func(hashVal hash.Hash, length uint64, isNew bool)

type AddCallbackSetter interface {
	SetAddCallback(callback AddCallback)
}

type GarbageCollector func(bytesToDelete uint64) (
	bytesDeleted uint64, err error)

type GarbageCollectorSetter interface {
	SetGarbageCollector(gc GarbageCollector)
}

type ObjectGetter interface {
	GetObject(hashVal hash.Hash) (uint64, io.ReadCloser, error)
}

type ObjectsGetter interface {
	GetObjects(hashes []hash.Hash) (ObjectsReader, error)
}

type ObjectsReader interface {
	Close() error
	NextObject() (uint64, io.ReadCloser, error)
}

type ObjectServer interface {
	AddObject(reader io.Reader, length uint64, expectedHash *hash.Hash) (
		hash.Hash, bool, error)
	CheckObjects(hashes []hash.Hash) ([]uint64, error)
	ObjectGetter
	ObjectsGetter
}

type StashingObjectServer interface {
	CommitObject(hash.Hash) error
	DeleteStashedObject(hashVal hash.Hash) error
	ObjectServer
	StashOrVerifyObject(io.Reader, uint64, *hash.Hash) (
		hash.Hash, []byte, error)
}

func GetObject(objSrv ObjectServer, hashVal hash.Hash) (
	uint64, io.ReadCloser, error) {
	return getObject(objSrv, hashVal)
}
