package objectserver

import (
	"github.com/Symantec/Dominator/lib/hash"
	"io"
)

type ObjectServer interface {
	AddObject([]byte, *hash.Hash) (hash.Hash, error)
	CheckObject(hash.Hash) (bool, error)
	GetObjectReader(hash.Hash) (uint64, io.ReadCloser, error)
}
