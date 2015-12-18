package objectclient

import (
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/objectserver"
)

func (objClient *ObjectClient) addObjects(datas [][]byte,
	expectedHashes []*hash.Hash) ([]hash.Hash, error) {
	for _, data := range datas {
		if len(data) < 1 {
			return nil, errors.New("zero length object cannot be added")
		}
	}
	srpcClient, err := srpc.DialHTTP("tcp", objClient.address)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Error dialing\t%s\n", err.Error()))
	}
	defer srpcClient.Close()
	conn, err := srpcClient.Call("ObjectServer.AddObjects")
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	go sendRequests(conn, datas, expectedHashes)
	decoder := gob.NewDecoder(conn)
	hashes := make([]hash.Hash, 0, len(datas))
	for range datas {
		var reply objectserver.AddObjectResponse
		if err := decoder.Decode(&reply); err != nil {
			return nil, err
		}
		if reply.Error != nil {
			return nil, err
		}
		hashes = append(hashes, reply.Hash)
	}
	return hashes, nil
}

func sendRequests(conn *srpc.Conn, datas [][]byte,
	expectedHashes []*hash.Hash) {
	defer conn.Flush()
	encoder := gob.NewEncoder(conn)
	for index, data := range datas {
		var request objectserver.AddObjectRequest
		request.Length = uint64(len(data))
		request.ExpectedHash = expectedHashes[index]
		encoder.Encode(request)
		conn.Write(data)
	}
	var request objectserver.AddObjectRequest // Signal end of stream.
	encoder.Encode(request)
}
