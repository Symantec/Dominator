package hypervisor

import (
	"bytes"
	"errors"
	"net"
	"strings"
)

const consoleTypeUnknown = "UNKNOWN ConsoleType"
const stateUnknown = "UNKNOWN State"
const volumeFormatUnknown = "UNKNOWN VolumeFormat"

var (
	consoleTypeToText = map[ConsoleType]string{
		ConsoleNone:  "none",
		ConsoleDummy: "dummy",
		ConsoleVNC:   "vnc",
	}
	textToConsoleType map[string]ConsoleType

	stateToText = map[State]string{
		StateStarting:      "starting",
		StateRunning:       "running",
		StateFailedToStart: "failed to start",
		StateStopping:      "stopping",
		StateStopped:       "stopped",
		StateDestroying:    "destroying",
		StateMigrating:     "migrating",
		StateExporting:     "exporting",
	}
	textToState map[string]State

	volumeFormatToText = map[VolumeFormat]string{
		VolumeFormatRaw:   "raw",
		VolumeFormatQCOW2: "qcow2",
	}
	textToVolumeFormat map[string]VolumeFormat
)

func init() {
	textToConsoleType = make(map[string]ConsoleType, len(consoleTypeToText))
	for consoleType, text := range consoleTypeToText {
		textToConsoleType[text] = consoleType
	}
	textToState = make(map[string]State, len(stateToText))
	for state, text := range stateToText {
		textToState[text] = state
	}
	textToVolumeFormat = make(map[string]VolumeFormat, len(volumeFormatToText))
	for format, text := range volumeFormatToText {
		textToVolumeFormat[text] = format
	}
}

func ShrinkIP(netIP net.IP) net.IP {
	switch len(netIP) {
	case 4:
		return netIP
	case 16:
		if ip4 := netIP.To4(); ip4 == nil {
			return netIP
		} else {
			return ip4
		}
	default:
		return netIP
	}
}

func (left *Address) Equal(right *Address) bool {
	if !left.IpAddress.Equal(right.IpAddress) {
		return false
	}
	if left.MacAddress != right.MacAddress {
		return false
	}
	return true
}

func (address *Address) Set(value string) error {
	if split := strings.Split(value, ";"); len(split) != 2 {
		return errors.New("malformed address pair: " + value)
	} else if ip := net.ParseIP(split[1]); ip == nil {
		return errors.New("unable to parse IP: " + split[1])
	} else if ip4 := ip.To4(); ip4 == nil {
		return errors.New("address is not IPv4: " + split[1])
	} else {
		*address = Address{IpAddress: ip4, MacAddress: split[0]}
		return nil
	}
}

func (address *Address) Shrink() {
	address.IpAddress = ShrinkIP(address.IpAddress)
}

func (address *Address) String() string {
	return address.IpAddress.String() + ";" + address.MacAddress
}

func (al *AddressList) String() string {
	buffer := &bytes.Buffer{}
	buffer.WriteString(`"`)
	for index, address := range *al {
		buffer.WriteString(address.String())
		if index < len(*al)-1 {
			buffer.WriteString(",")
		}
	}
	buffer.WriteString(`"`)
	return buffer.String()
}

func (al *AddressList) Set(value string) error {
	newList := make(AddressList, 0)
	if value != "" {
		addressStrings := strings.Split(value, ",")
		for _, addressString := range addressStrings {
			var address Address
			if err := address.Set(addressString); err != nil {
				return err
			}
			newList = append(newList, address)
		}
	}
	*al = newList
	return nil
}

func stringSlicesEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index, leftString := range left {
		if leftString != right[index] {
			return false
		}
	}
	return true
}

func (consoleType *ConsoleType) CheckValid() error {
	if _, ok := consoleTypeToText[*consoleType]; !ok {
		return errors.New(consoleTypeUnknown)
	} else {
		return nil
	}
}

func (consoleType ConsoleType) MarshalText() ([]byte, error) {
	if text := consoleType.String(); text == consoleTypeUnknown {
		return nil, errors.New(text)
	} else {
		return []byte(text), nil
	}
}

func (consoleType *ConsoleType) Set(value string) error {
	if val, ok := textToConsoleType[value]; !ok {
		return errors.New(consoleTypeUnknown)
	} else {
		*consoleType = val
		return nil
	}
}

func (consoleType ConsoleType) String() string {
	if str, ok := consoleTypeToText[consoleType]; !ok {
		return consoleTypeUnknown
	} else {
		return str
	}
}

func (consoleType *ConsoleType) UnmarshalText(text []byte) error {
	txt := string(text)
	if val, ok := textToConsoleType[txt]; ok {
		*consoleType = val
		return nil
	} else {
		return errors.New("unknown ConsoleType: " + txt)
	}
}

func (state State) MarshalText() ([]byte, error) {
	if text := state.String(); text == stateUnknown {
		return nil, errors.New(text)
	} else {
		return []byte(text), nil
	}
}

func (state State) String() string {
	if text, ok := stateToText[state]; ok {
		return text
	} else {
		return stateUnknown
	}
}

func (state *State) UnmarshalText(text []byte) error {
	txt := string(text)
	if val, ok := textToState[txt]; ok {
		*state = val
		return nil
	} else {
		return errors.New("unknown State: " + txt)
	}
}

func (left *Subnet) Equal(right *Subnet) bool {
	if left.Id != right.Id {
		return false
	}
	if !left.IpGateway.Equal(right.IpGateway) {
		return false
	}
	if !left.IpMask.Equal(right.IpMask) {
		return false
	}
	if left.DomainName != right.DomainName {
		return false
	}
	if !IpListsEqual(left.DomainNameServers, right.DomainNameServers) {
		return false
	}
	if left.Manage != right.Manage {
		return false
	}
	if left.VlanId != right.VlanId {
		return false
	}
	if !stringSlicesEqual(left.AllowedGroups, right.AllowedGroups) {
		return false
	}
	if !stringSlicesEqual(left.AllowedUsers, right.AllowedUsers) {
		return false
	}
	return true
}

func IpListsEqual(left, right []net.IP) bool {
	if len(left) != len(right) {
		return false
	}
	for index, leftIP := range left {
		if !leftIP.Equal(right[index]) {
			return false
		}
	}
	return true
}

func (subnet *Subnet) Shrink() {
	subnet.IpGateway = ShrinkIP(subnet.IpGateway)
	subnet.IpMask = ShrinkIP(subnet.IpMask)
	for index, ipAddr := range subnet.DomainNameServers {
		subnet.DomainNameServers[index] = ShrinkIP(ipAddr)
	}
}

func (left *VmInfo) Equal(right *VmInfo) bool {
	if !left.Address.Equal(&right.Address) {
		return false
	}
	if left.ConsoleType != right.ConsoleType {
		return false
	}
	if left.DestroyProtection != right.DestroyProtection {
		return false
	}
	if left.DisableVirtIO != right.DisableVirtIO {
		return false
	}
	if left.Hostname != right.Hostname {
		return false
	}
	if left.ImageName != right.ImageName {
		return false
	}
	if left.ImageURL != right.ImageURL {
		return false
	}
	if left.MemoryInMiB != right.MemoryInMiB {
		return false
	}
	if left.MilliCPUs != right.MilliCPUs {
		return false
	}
	if !stringSlicesEqual(left.OwnerGroups, right.OwnerGroups) {
		return false
	}
	if !stringSlicesEqual(left.OwnerUsers, right.OwnerUsers) {
		return false
	}
	if left.SpreadVolumes != right.SpreadVolumes {
		return false
	}
	if left.State != right.State {
		return false
	}
	if !left.Tags.Equal(right.Tags) {
		return false
	}
	if len(left.SecondaryAddresses) != len(right.SecondaryAddresses) {
		return false
	}
	for index, leftAddress := range left.SecondaryAddresses {
		if !leftAddress.Equal(&right.SecondaryAddresses[index]) {
			return false
		}
	}
	if !stringSlicesEqual(left.SecondarySubnetIDs, right.SecondarySubnetIDs) {
		return false
	}
	if left.SubnetId != right.SubnetId {
		return false
	}
	if left.Uncommitted != right.Uncommitted {
		return false
	}
	if len(left.Volumes) != len(right.Volumes) {
		return false
	}
	for index, leftVolume := range left.Volumes {
		if leftVolume != right.Volumes[index] {
			return false
		}
	}
	return true
}

func (volumeFormat VolumeFormat) MarshalText() ([]byte, error) {
	if text := volumeFormat.String(); text == volumeFormatUnknown {
		return nil, errors.New(text)
	} else {
		return []byte(text), nil
	}
}

func (volumeFormat VolumeFormat) String() string {
	if text, ok := volumeFormatToText[volumeFormat]; ok {
		return text
	} else {
		return volumeFormatUnknown
	}
}

func (volumeFormat *VolumeFormat) UnmarshalText(text []byte) error {
	txt := string(text)
	if val, ok := textToVolumeFormat[txt]; ok {
		*volumeFormat = val
		return nil
	} else {
		return errors.New("unknown VolumeFormat: " + txt)
	}
}
