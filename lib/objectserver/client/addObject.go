package client

import (
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/objectserver"
	"io"
)

func (objClient *ObjectClient) addObject(reader io.Reader, length uint64,
	expectedHash *hash.Hash) (hash.Hash, bool, error) {
	var request objectserver.AddObjectRequest
	var reply objectserver.AddObjectResponse
	if length < 1 {
		return reply.Hash, false, errors.New(
			"zero length object cannot be added")
	}
	srpcClient, err := srpc.DialHTTP("tcp", objClient.address, 0)
	if err != nil {
		return reply.Hash, false, errors.New(
			fmt.Sprintf("Error dialing\t%s\n", err.Error()))
	}
	defer srpcClient.Close()
	conn, err := srpcClient.Call("ObjectServer.AddObjects")
	if err != nil {
		return reply.Hash, false, err
	}
	defer conn.Close()
	request.Length = length
	request.ExpectedHash = expectedHash
	encoder := gob.NewEncoder(conn)
	encoder.Encode(request)
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
	encoder.Encode(request)
	conn.Flush()
	decoder := gob.NewDecoder(conn)
	if err := decoder.Decode(&reply); err != nil {
		return reply.Hash, false, err
	}
	if reply.Error != nil {
		return reply.Hash, false, err
	}
	if expectedHash != nil && *expectedHash != reply.Hash {
		return reply.Hash, false, errors.New(fmt.Sprintf(
			"received hash: %x != expected: %x",
			reply.Hash, *expectedHash))
	}
	return reply.Hash, reply.Added, nil
}
