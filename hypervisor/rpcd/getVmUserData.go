package rpcd

import (
	"io"
	"os"

	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
)

func (t *srpcType) GetVmUserData(conn *srpc.Conn, decoder srpc.Decoder,
	encoder srpc.Encoder) error {
	var request proto.GetVmUserDataRequest
	if err := decoder.Decode(&request); err != nil {
		return err
	}
	rc, length, err := t.manager.GetVmUserDataRPC(request.IpAddress,
		conn.GetAuthInformation(), request.AccessToken)
	if err != nil {
		if os.IsNotExist(err) {
			return encoder.Encode(proto.GetVmUserDataResponse{})
		}
		return encoder.Encode(proto.GetVmUserDataResponse{Error: err.Error()})
	}
	response := proto.GetVmUserDataResponse{Length: length}
	if err := encoder.Encode(response); err != nil {
		return err
	}
	_, err = io.CopyN(conn, rc, int64(length))
	return err
}
