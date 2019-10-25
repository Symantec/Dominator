package client

import (
	"errors"
	"fmt"
	"io"

	"github.com/Cloud-Foundations/Dominator/lib/hash"
	"github.com/Cloud-Foundations/Dominator/proto/objectserver"
)

func (objClient *ObjectClient) addObject(reader io.Reader, length uint64,
	expectedHash *hash.Hash) (hash.Hash, bool, error) {
	var request objectserver.AddObjectRequest
	var reply objectserver.AddObjectResponse
	if length < 1 {
		return reply.Hash, false, errors.New(
			"zero length object cannot be added")
	}
	srpcClient, err := objClient.getClient()
	if err != nil {
		return reply.Hash, false, err
	}
	conn, err := srpcClient.Call("ObjectServer.AddObjects")
	if err != nil {
		return reply.Hash, false, err
	}
	defer conn.Close()
	request.Length = length
	request.ExpectedHash = expectedHash
	conn.Encode(request)
	nCopied, err := io.Copy(conn, reader)
	if err != nil {
		return reply.Hash, false, err
	}
	if uint64(nCopied) != length {
		return reply.Hash, false, errors.New(fmt.Sprintf(
			"failed to copy, wanted: %d, got: %d bytes", length, nCopied))
	}
	// Send end-of-stream marker.
	request = objectserver.AddObjectRequest{}
	conn.Encode(request)
	conn.Flush()
	if err := conn.Decode(&reply); err != nil {
		return reply.Hash, false, err
	}
	if reply.ErrorString != "" {
		return reply.Hash, false, errors.New(reply.ErrorString)
	}
	if expectedHash != nil && *expectedHash != reply.Hash {
		return reply.Hash, false, errors.New(fmt.Sprintf(
			"received hash: %x != expected: %x",
			reply.Hash, *expectedHash))
	}
	return reply.Hash, reply.Added, nil
}
