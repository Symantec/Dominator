package hypervisor

import (
	"net"
	"time"
)

const (
	StateStarting      = 0
	StateRunning       = 1
	StateFailedToStart = 2
	StateStopping      = 3
	StateStopped       = 4
	StateDestroying    = 5
)

type AcknowledgeVmRequest struct {
	IpAddress net.IP
}

type AcknowledgeVmResponse struct {
	Error string
}

type AddAddressesToPoolRequest struct {
	Addresses []Address
}

type AddAddressesToPoolResponse struct {
	Error string
}

type Address struct {
	IpAddress  net.IP `json:",omitempty"`
	MacAddress string
}

type AddSubnetsRequest struct {
	Subnets []Subnet
}

type AddSubnetsResponse struct {
	Error string
}

type ChangeVmOwnerUsersRequest struct {
	IpAddress  net.IP
	OwnerUsers []string
}

type ChangeVmOwnerUsersResponse struct {
	Error string
}

type ChangeVmTagsRequest struct {
	IpAddress net.IP
	Tags      map[string]string
}

type ChangeVmTagsResponse struct {
	Error string
}

type CreateVmRequest struct {
	DhcpTimeout      time.Duration
	ImageDataSize    uint64
	ImageTimeout     time.Duration
	MinimumFreeBytes uint64
	RoundupPower     uint64
	SecondaryVolumes []Volume
	UserDataSize     uint64
	VmInfo
} // RAW image data (length=ImageDataSize) and user data (length=UserDataSize)
// are streamed afterwards.

type CreateVmResponse struct { // Multiple responses are sent.
	DhcpTimedOut    bool
	Final           bool // If true, this is the final response.
	IpAddress       net.IP
	ProgressMessage string
	Error           string
}

type DestroyVmRequest struct {
	IpAddress net.IP
}

type DestroyVmResponse struct {
	Error string
}

type DiscardVmOldImageRequest struct {
	IpAddress net.IP
}

type DiscardVmOldImageResponse struct {
	Error string
}

type DiscardVmOldUserDataRequest struct {
	IpAddress net.IP
}

type DiscardVmOldUserDataResponse struct {
	Error string
}

// The GetUpdates() RPC is fully streamed.
// The client sends no information to the server.
// The server sends a stream of Update messages.

type Update struct {
	AddressPool []Address          `json:",omitempty"`
	Subnets     []Subnet           `json:",omitempty"`
	VMs         map[string]*VmInfo `json:",omitempty"` // Key: IP address.
}

type GetVmInfoRequest struct {
	IpAddress net.IP
}

type GetVmInfoResponse struct {
	VmInfo VmInfo
	Error  string
}

type ListVMsRequest struct {
	Sort bool
}

type ListVMsResponse struct {
	IpAddresses []net.IP
}

type ReplaceVmImageRequest struct {
	DhcpTimeout      time.Duration
	ImageDataSize    uint64
	ImageName        string `json:",omitempty"`
	ImageTimeout     time.Duration
	ImageURL         string `json:",omitempty"`
	IpAddress        net.IP
	MinimumFreeBytes uint64
	RoundupPower     uint64
} // RAW image data (length=ImageDataSize) is streamed afterwards.

type ReplaceVmImageResponse struct { // Multiple responses are sent.
	DhcpTimedOut    bool
	Final           bool // If true, this is the final response.
	ProgressMessage string
	Error           string
}

type ReplaceVmUserDataRequest struct {
	IpAddress net.IP
	Size      uint64
} // User data (length=Size) are streamed afterwards.

type ReplaceVmUserDataResponse struct {
	Error string
}

type RestoreVmImageRequest struct {
	IpAddress net.IP
}

type RestoreVmImageResponse struct {
	Error string
}

type RestoreVmUserDataRequest struct {
	IpAddress net.IP
}

type RestoreVmUserDataResponse struct {
	Error string
}

type StartVmRequest struct {
	DhcpTimeout time.Duration
	IpAddress   net.IP
}

type StartVmResponse struct {
	DhcpTimedOut bool
	Error        string
}

type StopVmRequest struct {
	IpAddress net.IP
}

type StopVmResponse struct {
	Error string
}

type State uint

type Subnet struct {
	Id                string
	IpGateway         net.IP
	IpMask            net.IP // net.IPMask can't be JSON {en,de}coded.
	DomainNameServers []net.IP
}

type VmInfo struct {
	Address       Address
	ImageName     string `json:",omitempty"`
	ImageURL      string `json:",omitempty"`
	MemoryInMiB   uint64
	MilliCPUs     uint
	OwnerGroups   []string `json:",omitempty"`
	OwnerUsers    []string
	SpreadVolumes bool
	State         State
	Tags          map[string]string `json:",omitempty"`
	SubnetId      string
	Volumes       []Volume
}

type Volume struct {
	Size uint64
}
