package objectserver

import (
	"github.com/Symantec/Dominator/lib/hash"
	"io"
)

type ObjectsReader interface {
	NextObject() (uint64, io.ReadCloser, error)
}

type ObjectServer interface {
	AddObjects(datas [][]byte, expectedHashes []*hash.Hash) ([]hash.Hash, error)
	CheckObjects(hashes []hash.Hash) ([]bool, error)
	GetObjects(hashes []hash.Hash) (ObjectsReader, error)
}
