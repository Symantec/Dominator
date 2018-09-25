package rsync

// The GetBlocks() protocol is fully streamed.
// The client sends a GetBlocksRequest to the server.
// The server sends a stream of Block messages.

type GetBlocksRequest struct {
	BlockOrder uint8  // Valid values: 9-32.
	NumBlocks  uint64 // If zero: send a single block for the whole volume.
} // lib/hash.Hash messages are streamed afterwards.

type Block struct {
	Error string
	Index uint64
	Size  uint64 // If zero: no more blocks coming.
} // Block data are streamed afterwards.
