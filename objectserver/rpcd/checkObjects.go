package rpcd

import (
	"encoding/gob"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/objectserver"
)

func (t *srpcType) CheckObjects(conn *srpc.Conn) error {
	var request objectserver.CheckObjectsRequest
	var response objectserver.CheckObjectsResponse
	decoder := gob.NewDecoder(conn)
	if err := decoder.Decode(&request); err != nil {
		_, err = conn.WriteString(err.Error() + "\n")
		return err
	}
	var err error
	response.ObjectSizes, err = t.objectServer.CheckObjects(request.Hashes)
	if err != nil {
		_, err = conn.WriteString(err.Error() + "\n")
		return err
	}
	if _, err := conn.WriteString("\n"); err != nil {
		return err
	}
	return gob.NewEncoder(conn).Encode(response)
}

func (t *rpcType) CheckObjects(
	request objectserver.CheckObjectsRequest,
	reply *objectserver.CheckObjectsResponse) error {
	var response objectserver.CheckObjectsResponse
	var err error
	response.ObjectSizes, err = t.objectServer.CheckObjects(request.Hashes)
	if err != nil {
		return err
	}
	*reply = response
	return nil
}
