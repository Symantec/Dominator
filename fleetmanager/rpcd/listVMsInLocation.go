package rpcd

import (
	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/fleetmanager"
)

func (t *srpcType) ListVMsInLocation(conn *srpc.Conn, decoder srpc.Decoder,
	encoder srpc.Encoder) error {
	var request proto.ListVMsInLocationRequest
	if err := decoder.Decode(&request); err != nil {
		return err
	}
	addresses, err := t.hypervisorsManager.ListVMsInLocation(
		request.Location)
	if err != nil {
		response := proto.ListVMsInLocationResponse{
			Error: errors.ErrorToString(err),
		}
		if err := encoder.Encode(response); err != nil {
			return err
		}
		return nil
	}
	// TODO(rgooch): Chunk the response.
	response := proto.ListVMsInLocationResponse{IpAddresses: addresses}
	if err := encoder.Encode(response); err != nil {
		return err
	}
	response.IpAddresses = nil // Send end-of-chunks message.
	return encoder.Encode(response)
}
