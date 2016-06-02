package objectserver

import (
	"github.com/Symantec/Dominator/lib/hash"
)

// The AddObjects() RPC requires the client to send a stream of AddObjectRequest
// objects in Gob format. To signify the end of the stream, the client should
// send an AddObjectRequest object with .Length == 0.
// The server will send one AddObjectResponse for each AddObjectRequest, but it
// will not flush the connection until the client signals the end of the stream.
type AddObjectRequest struct {
	Length       uint64
	ExpectedHash *hash.Hash
} // Object data are streamed afterwards.

type AddObjectResponse struct {
	Error error
	Hash  hash.Hash
	Added bool // If true: object was added, else object already existed.
}

type CheckObjectsRequest struct {
	Hashes []hash.Hash
}

type CheckObjectsResponse struct {
	ObjectSizes []uint64 // size == 0: object not found.
}

// This is used in the special GetObjects streaming HTTP/RPC protocol.
type GetObjectsRequest struct {
	Exclusive bool // For initial performance benchmarking only.
	Hashes    []hash.Hash
}

type GetObjectsResponse struct {
	ResponseString string
	ObjectSizes    []uint64
} // Object datas are streamed afterwards.
