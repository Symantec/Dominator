package rpcd

import (
	"encoding/gob"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/objectserver"
	"runtime"
)

func (t *srpcType) AddObjects(conn *srpc.Conn) {
	defer conn.Flush()
	var request objectserver.AddObjectsRequest
	var response objectserver.AddObjectsResponse
	decoder := gob.NewDecoder(conn)
	if err := decoder.Decode(&request); err != nil {
		conn.WriteString(err.Error() + "\n")
		return
	}
	if err := t.addObjects(request, &response); err != nil {
		conn.WriteString(err.Error() + "\n")
		return
	}
	conn.WriteString("\n")
	encoder := gob.NewEncoder(conn)
	encoder.Encode(response)
}

func (t *srpcType) addObjects(request objectserver.AddObjectsRequest,
	reply *objectserver.AddObjectsResponse) error {
	var response objectserver.AddObjectsResponse
	var err error
	response.Hashes, err = t.objectServer.AddObjects(request.ObjectDatas,
		request.ExpectedHashes)
	if err != nil {
		return err
	}
	*reply = response
	runtime.GC() // An opportune time to take out the garbage.
	return nil
}
