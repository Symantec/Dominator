package objectserver

import (
	"github.com/Symantec/Dominator/lib/hash"
	"io"
)

type ObjectServer interface {
	AddObject([]byte, *hash.Hash) (hash.Hash, error)
	CheckObject(hash.Hash) bool
	GetObjectReader(hash.Hash) (uint64, io.Reader, error)
}
