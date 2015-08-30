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
	ObjectSizes []uint64 // size == 0: object not found.
}

// This is used in the special GetObjects streaming HTTP/RPC protocol.
type GetObjectsRequest struct {
	Hashes []hash.Hash
}

type GetObjectsResponse struct {
	ResponseString string
	ObjectSizes    []uint64
} // Object datas are streamed afterwards.
