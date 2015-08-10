package objectserver

import (
	"github.com/Symantec/Dominator/lib/hash"
	"io"
)

type ObjectServer interface {
	CheckObject(hash.Hash) bool
	GetObjectReader(hash.Hash) (uint64, io.Reader, error)
	PutObject(uint64, []byte) (hash.Hash, error)
}
