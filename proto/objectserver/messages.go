package objectserver

import (
	"github.com/Symantec/Dominator/lib/hash"
)

type AddObjectSubrequest struct {
	ObjectData   []byte
	ExpectedHash *hash.Hash
}

type AddObjectsRequest struct {
	ObjectsToAdd []*AddObjectSubrequest
}

type AddObjectsResponse struct {
	Hashes []hash.Hash
}

type CheckObjectsRequest struct {
	Objects []hash.Hash
}

type CheckObjectsResponse struct {
	ObjectsPresent []bool
}

type GetObjectsRequest struct {
	Objects []hash.Hash
}

type GetObjectsResponse struct {
	ObjectSizes []uint64
}
