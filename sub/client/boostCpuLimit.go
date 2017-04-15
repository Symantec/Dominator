package client

import (
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/sub"
)

func boostCpuLimit(client *srpc.Client) error {
	request := sub.BoostCpuLimitRequest{}
	var reply sub.BoostCpuLimitResponse
	return client.RequestReply("Subd.BoostCpuLimit", request, &reply)
}
