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

type ListHypervisorLocationsRequest struct {
	TopLocation string
}

type ListHypervisorLocationsResponse struct {
	Locations []string
	Error     string
}

type ListHypervisorsInLocationRequest struct {
	Location string
}

type ListHypervisorsInLocationResponse struct {
	HypervisorAddresses []string // host:port
	Error               string
}

type ListVMsInLocationRequest struct {
	Location string
}

// A stream of ListVMsInLocationResponse messages is sent, until either the
// length of the IpAddresses field is zero, or the Error field != "".
type ListVMsInLocationResponse struct {
	IpAddresses []net.IP
	Error       string
}
