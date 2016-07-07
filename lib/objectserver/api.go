package objectserver

import (
	"github.com/Symantec/Dominator/lib/hash"
	"io"
)

type ObjectsReader interface {
	Close() error
	NextObject() (uint64, io.ReadCloser, error)
}

type ObjectGetter interface {
	GetObject(hashVal hash.Hash) (uint64, io.ReadCloser, error)
}

type ObjectServer interface {
	AddObject(reader io.Reader, length uint64, expectedHash *hash.Hash) (
		hash.Hash, bool, error)
	CheckObjects(hashes []hash.Hash) ([]uint64, error)
	ObjectGetter
	GetObjects(hashes []hash.Hash) (ObjectsReader, error)
}

func GetObject(objSrv ObjectServer, hashVal hash.Hash) (
	uint64, io.ReadCloser, error) {
	return getObject(objSrv, hashVal)
}
