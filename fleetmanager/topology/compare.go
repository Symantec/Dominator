package topology

import (
	"net"
)

func (left *Topology) equal(right *Topology) bool {
	if left == nil || right == nil {
		return false
	}
	if len(left.machineParents) != len(right.machineParents) {
		return false
	}
	return left.Root.equal(right.Root)
}

func (left *Directory) equal(right *Directory) bool {
	if left.Name != right.Name {
		return false
	}
	if len(left.Directories) != len(right.Directories) {
		return false
	}
	if len(left.Machines) != len(right.Machines) {
		return false
	}
	if len(left.Subnets) != len(right.Subnets) {
		return false
	}
	for index, leftSubdir := range left.Directories {
		if !leftSubdir.equal(right.Directories[index]) {
			return false
		}
	}
	for index, leftMachine := range left.Machines {
		if !leftMachine.equal(right.Machines[index]) {
			return false
		}
	}
	for index, leftSubnet := range left.Subnets {
		if !leftSubnet.equal(right.Subnets[index]) {
			return false
		}
	}
	return true
}

func (left *Machine) equal(right *Machine) bool {
	if left.Hostname != right.Hostname {
		return false
	}
	if !left.HostIpAddress.Equal(right.HostIpAddress) {
		return false
	}
	if left.HostMacAddress.String() != right.HostMacAddress.String() {
		return false
	}
	return true
}

func (left *Subnet) equal(right *Subnet) bool {
	if left.Id != right.Id {
		return false
	}
	if !left.IpGateway.Equal(right.IpGateway) {
		return false
	}
	if !left.IpMask.Equal(right.IpMask) {
		return false
	}
	if !ipListsEqual(left.DomainNameServers, right.DomainNameServers) {
		return false
	}
	return ipListsEqual(left.ReservedIPs, right.ReservedIPs)
}

func ipListsEqual(left, right []net.IP) bool {
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
