package client

import (
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/sub"
)

func boostCpuLimit(client *srpc.Client) error {
	request := sub.BoostCpuLimitRequest{}
	var reply sub.BoostCpuLimitResponse
	return client.RequestReply("Subd.BoostCpuLimit", request, &reply)
}
