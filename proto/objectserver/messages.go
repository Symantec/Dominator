package objectserver

import (
	"github.com/Symantec/Dominator/lib/hash"
)

type AddObjectsRequest struct {
	ObjectDatas    [][]byte
	ExpectedHashes []*hash.Hash
}

type AddObjectsResponse struct {
	Hashes []hash.Hash
}

type CheckObjectsRequest struct {
	Hashes []hash.Hash
}

type CheckObjectsResponse struct {
	ObjectsPresent []bool
}

type GetObjectsRequest struct {
	Hashes []hash.Hash
}

type GetObjectsResponse struct {
	ObjectSizes []uint64
	Objects     [][]byte
}
