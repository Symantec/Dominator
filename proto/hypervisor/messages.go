package hypervisor

import (
	"net"
	"time"

	"github.com/Symantec/Dominator/lib/tags"
)

const (
	ConsoleNone  = 0
	ConsoleDummy = 1
	ConsoleVNC   = 2

	StateStarting      = 0
	StateRunning       = 1
	StateFailedToStart = 2
	StateStopping      = 3
	StateStopped       = 4
	StateDestroying    = 5
	StateMigrating     = 6
	StateExporting     = 7

	VolumeFormatRaw   = 0
	VolumeFormatQCOW2 = 1
)

type AcknowledgeVmRequest struct {
	IpAddress net.IP
}

type AcknowledgeVmResponse struct {
	Error string
}

type Address struct {
	IpAddress  net.IP `json:",omitempty"`
	MacAddress string
}

type BecomePrimaryVmOwnerRequest struct {
	IpAddress net.IP
}

type BecomePrimaryVmOwnerResponse struct {
	Error string
}

type ChangeAddressPoolRequest struct {
	AddressesToAdd       []Address       // Will be added to free pool.
	AddressesToRemove    []Address       // Will be removed from free pool.
	MaximumFreeAddresses map[string]uint // Key: subnet ID.
}

type ChangeAddressPoolResponse struct {
	Error string
}

type ChangeOwnersRequest struct {
	OwnerGroups []string `json:",omitempty"`
	OwnerUsers  []string `json:",omitempty"`
}

type ChangeOwnersResponse struct {
	Error string
}

type ChangeVmConsoleTypeRequest struct {
	ConsoleType ConsoleType
	IpAddress   net.IP
}

type ChangeVmConsoleTypeResponse struct {
	Error string
}

type ChangeVmDestroyProtectionRequest struct {
	DestroyProtection bool
	IpAddress         net.IP
}

type ChangeVmDestroyProtectionResponse struct {
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
	Tags      tags.Tags
}

type ChangeVmTagsResponse struct {
	Error string
}

type CommitImportedVmRequest struct {
	IpAddress net.IP
}

type CommitImportedVmResponse struct {
	Error string
}

// The ConnectToVmConsole RPC is fully streamed. After the request/response,
// the connection/client is hijacked and each side of the connection will send
// a stream of bytes.
type ConnectToVmConsoleRequest struct {
	IpAddress net.IP
}

type ConnectToVmConsoleResponse struct {
	Error string
}

// The ConnectToVmSerialPort RPC is fully streamed. After the request/response,
// the connection/client is hijacked and each side of the connection will send
// a stream of bytes.
type ConnectToVmSerialPortRequest struct {
	IpAddress  net.IP
	PortNumber uint
}

type ConnectToVmSerialPortResponse struct {
	Error string
}

type ConsoleType uint

type CopyVmRequest struct {
	AccessToken      []byte
	IpAddress        net.IP
	SourceHypervisor string
	VmInfo
}

type CopyVmResponse struct { // Multiple responses are sent.
	Error           string
	Final           bool // If true, this is the final response.
	IpAddress       net.IP
	ProgressMessage string
}

type CreateVmRequest struct {
	DhcpTimeout      time.Duration
	ImageDataSize    uint64
	ImageTimeout     time.Duration
	MinimumFreeBytes uint64
	RoundupPower     uint64
	SecondaryVolumes []Volume
	SkipBootloader   bool
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

type DeleteVmVolumeRequest struct {
	AccessToken []byte
	IpAddress   net.IP
	VolumeIndex uint
}

type DeleteVmVolumeResponse struct {
	Error string
}

type DestroyVmRequest struct {
	AccessToken []byte
	IpAddress   net.IP
}

type DestroyVmResponse struct {
	Error string
}

type DiscardVmAccessTokenRequest struct {
	AccessToken []byte
	IpAddress   net.IP
}

type DiscardVmAccessTokenResponse struct {
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

type DiscardVmSnapshotRequest struct {
	IpAddress net.IP
}

type DiscardVmSnapshotResponse struct {
	Error string
}

type ExportLocalVmInfo struct {
	Bridges []string
	LocalVmInfo
}

type ExportLocalVmRequest struct {
	IpAddress          net.IP
	VerificationCookie []byte `json:",omitempty"`
}

type ExportLocalVmResponse struct {
	Error  string
	VmInfo ExportLocalVmInfo
}

type GetRootCookiePathRequest struct{}

type GetRootCookiePathResponse struct {
	Error string
	Path  string
}

// The GetUpdates() RPC is fully streamed.
// The client may or may not send GetUpdateRequest messages to the server.
// The server sends a stream of Update messages.

type GetUpdateRequest struct{}

type Update struct {
	HaveAddressPool  bool               `json:",omitempty"`
	AddressPool      []Address          `json:",omitempty"` // Used & free.
	NumFreeAddresses map[string]uint    `json:",omitempty"` // Key: subnet ID.
	HealthStatus     string             `json:",omitempty"`
	HaveSerialNumber bool               `json:",omitempty"`
	SerialNumber     string             `json:",omitempty"`
	HaveSubnets      bool               `json:",omitempty"`
	Subnets          []Subnet           `json:",omitempty"`
	HaveVMs          bool               `json:",omitempty"`
	VMs              map[string]*VmInfo `json:",omitempty"` // Key: IP address.
}

type GetVmAccessTokenRequest struct {
	IpAddress net.IP
	Lifetime  time.Duration
}

type GetVmAccessTokenResponse struct {
	Token []byte `json:",omitempty"`
	Error string
}

type GetVmInfoRequest struct {
	IpAddress net.IP
}

type GetVmInfoResponse struct {
	VmInfo VmInfo
	Error  string
}

type GetVmUserDataRequest struct {
	AccessToken []byte
	IpAddress   net.IP
}

type GetVmUserDataResponse struct {
	Error  string
	Length uint64
} // Data (length=Length) are streamed afterwards.

// The GetVmVolume() RPC is followed by the proto/rsync.GetBlocks message.

type GetVmVolumeRequest struct {
	AccessToken []byte
	IpAddress   net.IP
	VolumeIndex uint
}

type GetVmVolumeResponse struct {
	Error string
}

type ImportLocalVmRequest struct {
	VerificationCookie []byte `json:",omitempty"`
	VmInfo
	VolumeFilenames []string
}

type ImportLocalVmResponse struct {
	Error string
}

type ListVMsRequest struct {
	OwnerUsers []string
	Sort       bool
}

type ListVMsResponse struct {
	IpAddresses []net.IP
}

type ListVolumeDirectoriesRequest struct{}

type ListVolumeDirectoriesResponse struct {
	Directories []string
	Error       string
}

type LocalVolume struct {
	DirectoryToCleanup string
	Filename           string
}

type LocalVmInfo struct {
	VmInfo
	VolumeLocations []LocalVolume
}

type MigrateVmRequest struct {
	AccessToken      []byte
	DhcpTimeout      time.Duration
	IpAddress        net.IP
	SourceHypervisor string
}

type MigrateVmResponse struct { // Multiple responses are sent.
	Error           string
	Final           bool // If true, this is the final response.
	ProgressMessage string
	RequestCommit   bool
}

type MigrateVmResponseResponse struct {
	Commit bool
}

type NetbootMachineRequest struct {
	Address                      Address
	Files                        map[string][]byte
	FilesExpiration              time.Duration
	Hostname                     string
	NumAcknowledgementsToWaitFor uint
	OfferExpiration              time.Duration
	Subnet                       *Subnet
	WaitTimeout                  time.Duration
}

type NetbootMachineResponse struct {
	Error string
}

type PatchVmImageRequest struct {
	ImageName    string
	ImageTimeout time.Duration
	IpAddress    net.IP
}

type PatchVmImageResponse struct { // Multiple responses are sent.
	Final           bool // If true, this is the final response.
	ProgressMessage string
	Error           string
}

type PrepareVmForMigrationRequest struct {
	AccessToken []byte
	Enable      bool
	IpAddress   net.IP
}

type PrepareVmForMigrationResponse struct {
	Error string
}

type ProbeVmPortRequest struct {
	IpAddress  net.IP
	PortNumber uint
	Timeout    time.Duration
}

type ProbeVmPortResponse struct {
	PortIsOpen bool
	Error      string
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
	SkipBootloader   bool
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

type RestoreVmFromSnapshotRequest struct {
	IpAddress         net.IP
	ForceIfNotStopped bool
}

type RestoreVmFromSnapshotResponse struct {
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

type SnapshotVmRequest struct {
	IpAddress         net.IP
	ForceIfNotStopped bool
	RootOnly          bool
}

type SnapshotVmResponse struct {
	Error string
}

type StartVmRequest struct {
	AccessToken []byte
	DhcpTimeout time.Duration
	IpAddress   net.IP
}

type StartVmResponse struct {
	DhcpTimedOut bool
	Error        string
}

type StopVmRequest struct {
	AccessToken []byte
	IpAddress   net.IP
}

type StopVmResponse struct {
	Error string
}

type State uint

type Subnet struct {
	Id                string
	IpGateway         net.IP
	IpMask            net.IP // net.IPMask can't be JSON {en,de}coded.
	DomainName        string `json:",omitempty"`
	DomainNameServers []net.IP
	Manage            bool     `json:",omitempty"`
	VlanId            uint     `json:",omitempty"`
	AllowedGroups     []string `json:",omitempty"`
	AllowedUsers      []string `json:",omitempty"`
}

type TraceVmMetadataRequest struct {
	IpAddress net.IP
}

type TraceVmMetadataResponse struct {
	Error string
} // A stream of strings (trace paths) follow.

type UpdateSubnetsRequest struct {
	Add    []Subnet
	Change []Subnet
	Delete []string
}

type UpdateSubnetsResponse struct {
	Error string
}

type VmInfo struct {
	Address            Address
	ConsoleType        ConsoleType `json:",omitempty"`
	DestroyProtection  bool        `json:",omitempty"`
	DisableVirtIO      bool        `json:",omitempty"`
	Hostname           string      `json:",omitempty"`
	ImageName          string      `json:",omitempty"`
	ImageURL           string      `json:",omitempty"`
	MemoryInMiB        uint64
	MilliCPUs          uint
	OwnerGroups        []string `json:",omitempty"`
	OwnerUsers         []string `json:",omitempty"`
	SpreadVolumes      bool     `json:",omitempty"`
	State              State
	Tags               tags.Tags `json:",omitempty"`
	SecondaryAddresses []Address `json:",omitempty"`
	SecondarySubnetIDs []string  `json:",omitempty"`
	SubnetId           string    `json:",omitempty"`
	Uncommitted        bool      `json:",omitempty"`
	Volumes            []Volume  `json:",omitempty"`
}

type Volume struct {
	Size   uint64
	Format VolumeFormat
}

type VolumeFormat uint
