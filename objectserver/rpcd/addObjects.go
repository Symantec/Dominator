package rpcd

import (
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/objectserver"
	"io"
	"runtime"
)

func (t *srpcType) AddObjects(conn *srpc.Conn) {
	defer runtime.GC() // An opportune time to take out the garbage.
	defer conn.Flush()
	decoder := gob.NewDecoder(conn)
	encoder := gob.NewEncoder(conn)
	for {
		var request objectserver.AddObjectRequest
		var response objectserver.AddObjectResponse
		if err := decoder.Decode(&request); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				return
			}
			response.Error = err
		} else if request.Length < 1 {
			return
		} else {
			response.Error = t.addObject(conn, request, &response)
		}
		encoder.Encode(response)
		if response.Error != nil {
			return
		}
	}
}

func (t *srpcType) addObject(conn *srpc.Conn,
	request objectserver.AddObjectRequest,
	response *objectserver.AddObjectResponse) error {
	data := make([]byte, request.Length)
	nRead, err := io.ReadFull(conn, data)
	if err != nil {
		return err
	}
	if uint64(nRead) != request.Length {
		return errors.New(fmt.Sprintf(
			"failed to read data, wanted: %d, got: %d bytes",
			request.Length, nRead))
	}
	datas := [][]byte{data}
	expectedHashes := []*hash.Hash{request.ExpectedHash}
	hashes, err := t.objectServer.AddObjects(datas, expectedHashes)
	if err != nil {
		return err
	}
	response.Hash = hashes[0]
	return nil
}
