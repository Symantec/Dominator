package rpcd

import (
	"io"
	"os"

	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
)

func (t *srpcType) GetVmUserData(conn *srpc.Conn) error {
	var request proto.GetVmUserDataRequest
	if err := conn.Decode(&request); err != nil {
		return err
	}
	rc, length, err := t.manager.GetVmUserDataRPC(request.IpAddress,
		conn.GetAuthInformation(), request.AccessToken)
	if err != nil {
		if os.IsNotExist(err) {
			return conn.Encode(proto.GetVmUserDataResponse{})
		}
		return conn.Encode(proto.GetVmUserDataResponse{Error: err.Error()})
	}
	response := proto.GetVmUserDataResponse{Length: length}
	if err := conn.Encode(response); err != nil {
		return err
	}
	_, err = io.CopyN(conn, rc, int64(length))
	return err
}
