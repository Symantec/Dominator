package objectserver

import (
	"github.com/Cloud-Foundations/Dominator/lib/hash"
	"time"
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
	ErrorString string
	Hash        hash.Hash
	Added       bool // If true: object was added, else object already existed.
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

type TestBandwidthRequest struct {
	Duration     time.Duration // Ignored when sending to server.
	ChunkSize    uint          // Maximum permitted: 65535.
	SendToServer bool
} // The transmitter sends chunks of random data with a marker byte after each.
// If the marker byte is zero, no more chunks are sent and the response
// message is sent.

type TestBandwidthResponse struct {
	ServerDuration time.Duration
}
