package fleetmanager

import (
	"net"

	"github.com/Symantec/Dominator/lib/tags"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
)

type GetHypervisorForVMRequest struct {
	IpAddress net.IP
}

type GetHypervisorForVMResponse struct {
	HypervisorAddress string // host:port
	Error             string
}

// The GetUpdates() RPC is fully streamed.
// The client sends a single GetUpdatesRequest message.
// The server sends a stream of Update messages.

type GetUpdatesRequest struct {
	Location string
}

type Update struct {
	ChangedMachines []*Machine               `json:",omitempty"`
	ChangedVMs      map[string]*proto.VmInfo `json:",omitempty"` // Key: IPaddr
	DeletedMachines []string                 `json:",omitempty"` // Hostname
	DeletedVMs      []string                 `json:",omitempty"` // IPaddr
}

type HardwareAddr net.HardwareAddr

type ListHypervisorLocationsRequest struct {
	TopLocation string
}

type ListHypervisorLocationsResponse struct {
	Locations []string
	Error     string
}

type ListHypervisorsInLocationRequest struct {
	Location string
	SubnetId string
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

type Machine struct {
	Hostname       string       `json:",omitempty"`
	HostIpAddress  net.IP       `json:",omitempty"`
	HostMacAddress HardwareAddr `json:",omitempty"`
	Tags           tags.Tags    `json:",omitempty"`
}
