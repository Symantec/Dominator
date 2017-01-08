package client

import (
	"github.com/Symantec/Dominator/lib/srpc"
)

func AddDevice(client *srpc.Client, deviceId string, adder func() error) error {
	return addDevice(client, deviceId, adder)
}
