package client

import (
	"github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	proto "github.com/Cloud-Foundations/Dominator/proto/fleetmanager"
)

func powerOnMachine(client *srpc.Client, hostname string) error {
	request := proto.PowerOnMachineRequest{Hostname: hostname}
	var reply proto.PowerOnMachineResponse
	err := client.RequestReply("FleetManager.PowerOnMachine", request, &reply)
	if err != nil {
		return err
	}
	return errors.New(reply.Error)
}
