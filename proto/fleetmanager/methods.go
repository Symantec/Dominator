package fleetmanager

import (
	"bytes"
	"errors"
	"net"
)

func listsEqual(left, right []string) bool {
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

func parseMAC(text []byte) (HardwareAddr, error) {
	text = bytes.ToLower(text)
	buf := make([]byte, 20)
	writePosition := 0
	for _, char := range text {
		var value byte
		if char >= '0' && char <= '9' {
			value = char - '0'
		} else if char >= 'a' && char <= 'f' {
			value = 10 + char - 'a'
		} else if char == ':' {
			writePosition++
			if writePosition >= len(buf) {
				return nil, errors.New("invalid MAC")
			}
			continue
		} else {
			return nil, errors.New("invalid MAC")
		}
		if buf[writePosition]&0xf0 != 0 {
			return nil, errors.New("invalid MAC")
		}
		buf[writePosition] = buf[writePosition]<<4 + value
	}
	addr := make([]byte, writePosition+1) // Make a copy just long enough.
	copy(addr, buf)
	return addr, nil
}

func (left *Machine) Equal(right *Machine) bool {
	if !left.IPMI.Equal(&right.IPMI) {
		return false
	}
	if !left.NetworkEntry.Equal(&right.NetworkEntry) {
		return false
	}
	if !listsEqual(left.OwnerGroups, right.OwnerGroups) {
		return false
	}
	if !listsEqual(left.OwnerUsers, right.OwnerUsers) {
		return false
	}
	if len(left.SecondaryNetworkEntries) != len(right.SecondaryNetworkEntries) {
		return false
	}
	for index, leftNetworkEntry := range left.SecondaryNetworkEntries {
		if !leftNetworkEntry.Equal(&right.SecondaryNetworkEntries[index]) {
			return false
		}
	}
	if !left.Tags.Equal(right.Tags) {
		return false
	}
	return true
}

func (left *NetworkEntry) Equal(right *NetworkEntry) bool {
	if left.Hostname != right.Hostname {
		return false
	}
	if !left.HostIpAddress.Equal(right.HostIpAddress) {
		return false
	}
	if left.HostMacAddress.String() != right.HostMacAddress.String() {
		return false
	}
	if left.SubnetId != right.SubnetId {
		return false
	}
	return true
}

func (addr HardwareAddr) MarshalText() (text []byte, err error) {
	return []byte(addr.String()), nil
}

func (addr HardwareAddr) String() string {
	return net.HardwareAddr(addr).String()
}

func (addr *HardwareAddr) UnmarshalText(text []byte) error {
	text = bytes.ToLower(text)
	if newAddr, err := parseMAC(text); err == nil {
		*addr = newAddr
		return nil
	}
	if hw, err := net.ParseMAC(string(text)); err != nil {
		return err
	} else {
		*addr = HardwareAddr(hw)
		return nil
	}
}
