package objectserver

import (
	"github.com/Symantec/Dominator/lib/hash"
)

type AddObjectSubrequest struct {
	ObjectData   []byte
	ExpectedHash *hash.Hash
}

type AddFilesRequest struct {
	ObjectsToAdd []*AddObjectSubrequest
}

type AddFilesResponse struct {
	Hashes []hash.Hash
}

type GetFilesRequest struct {
	Objects []hash.Hash
}

type GetFilesResponse struct {
	ObjectSizes []uint64
}
