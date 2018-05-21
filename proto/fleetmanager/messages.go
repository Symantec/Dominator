package fleetmanager

import (
	"net"
)

type GetHypervisorForVMRequest struct {
	IpAddress net.IP
}

type GetHypervisorForVMResponse struct {
	HypervisorAddress string // host:port
	Error             string
}
