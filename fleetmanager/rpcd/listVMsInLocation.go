package rpcd

import (
	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/fleetmanager"
)

func (t *srpcType) ListVMsInLocation(conn *srpc.Conn) error {
	var request proto.ListVMsInLocationRequest
	if err := conn.Decode(&request); err != nil {
		return err
	}
	addresses, err := t.hypervisorsManager.ListVMsInLocation(
		request.Location)
	if err != nil {
		response := proto.ListVMsInLocationResponse{
			Error: errors.ErrorToString(err),
		}
		if err := conn.Encode(response); err != nil {
			return err
		}
		return nil
	}
	// TODO(rgooch): Chunk the response.
	response := proto.ListVMsInLocationResponse{IpAddresses: addresses}
	if err := conn.Encode(response); err != nil {
		return err
	}
	response.IpAddresses = nil // Send end-of-chunks message.
	return conn.Encode(response)
}
