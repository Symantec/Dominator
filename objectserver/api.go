package objectserver

import (
	"github.com/Symantec/Dominator/lib/hash"
	"io"
)

type ObjectServer interface {
	AddObjects([][]byte, []*hash.Hash) ([]hash.Hash, error)
	CheckObjects([]hash.Hash) ([]bool, error)
	GetObjectReader(hash.Hash) (uint64, io.ReadCloser, error)
}
