package objectserver

import (
	"github.com/Symantec/Dominator/lib/hash"
	"io"
)

type FullObjectServer interface {
	DeleteObject(hashVal hash.Hash) error
	ObjectServer
	ListObjectSizes() map[hash.Hash]uint64
	ListObjects() []hash.Hash
	NumObjects() uint64
}

type GarbageCollector func(bytesToDelete uint64) (
	bytesDeleted uint64, err error)

type GarbageCollectorSetter interface {
	SetGarbageCollector(gc GarbageCollector)
}

type ObjectGetter interface {
	GetObject(hashVal hash.Hash) (uint64, io.ReadCloser, error)
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
	GetObjects(hashes []hash.Hash) (ObjectsReader, error)
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
