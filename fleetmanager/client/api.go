package client

import (
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
)

func PowerOnMachine(client *srpc.Client, hostname string) error {
	return powerOnMachine(client, hostname)
}
