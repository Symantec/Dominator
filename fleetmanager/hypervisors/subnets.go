package hypervisors

import (
	"fmt"
	"net"

	"github.com/Symantec/Dominator/fleetmanager/topology"
)

func copyIp(ip net.IP) net.IP {
	retval := make(net.IP, len(ip))
	copy(retval, ip)
	return retval
}

func decrementIp(ip net.IP) {
	for index := len(ip) - 1; index >= 0; index-- {
		if ip[index] > 0 {
			ip[index]--
			return
		}
		ip[index] = 0xff
	}
}

func incrementIp(ip net.IP) {
	for index := len(ip) - 1; index >= 0; index-- {
		if ip[index] < 255 {
			ip[index]++
			return
		}
		ip[index] = 0
	}
}

func invertByte(input byte) byte {
	var inverted byte
	for index := 0; index < 8; index++ {
		inverted <<= 1
		if input&0x80 == 0 {
			inverted |= 1
		}
		input <<= 1
	}
	return inverted
}

// This must be called with the lock held.
func (m *Manager) checkIpReserved(tSubnet *topology.Subnet, ip net.IP) bool {
	if ip.Equal(tSubnet.IpGateway) {
		return true
	}
	ipAddr := ip.String()
	if tSubnet.CheckIfIpIsReserved(ipAddr) {
		return true
	}
	if _, ok := m.allocatingIPs[ipAddr]; ok {
		return true
	}
	if _, ok := m.migratingIPs[ipAddr]; ok {
		return true
	}
	return false
}

// This must be called with the lock held. This will update the allocatingIPs
// map.
func (m *Manager) findFreeIPs(tSubnet *topology.Subnet,
	numNeeded uint) ([]net.IP, error) {
	var freeIPs []net.IP
	gatewayIp := tSubnet.IpGateway.String()
	subnet, ok := m.subnets[gatewayIp]
	if !ok {
		return nil, fmt.Errorf("subnet for gateway: %s not found", gatewayIp)
	}
	initialIp := copyIp(subnet.nextIp)
	for numNeeded > 0 {
		if !m.checkIpReserved(subnet.subnet, subnet.nextIp) {
			registered, err := m.storer.CheckIpIsRegistered(subnet.nextIp)
			if err != nil {
				return nil, err
			}
			if !registered {
				freeIPs = append(freeIPs, copyIp(subnet.nextIp))
				numNeeded--
			}
		}
		incrementIp(subnet.nextIp)
		if subnet.nextIp.Equal(subnet.stopIp) {
			copy(subnet.nextIp, subnet.startIp)
		}
		if initialIp.Equal(subnet.nextIp) {
			break
		}
	}
	for _, ip := range freeIPs {
		m.allocatingIPs[ip.String()] = struct{}{}
	}
	return freeIPs, nil
}

func (m *Manager) initInvertTable() {
	for value := 0; value < 256; value++ {
		m.invertTable[value] = invertByte(byte(value))
	}
}

func (m *Manager) invertIP(input net.IP) net.IP {
	inverted := make(net.IP, len(input))
	for index, value := range input {
		inverted[index] = m.invertTable[value]
	}
	return inverted
}

func (m *Manager) makeSubnet(tSubnet *topology.Subnet) *subnetType {
	networkIp := tSubnet.IpGateway.Mask(net.IPMask(tSubnet.IpMask))
	var startIp, stopIp net.IP
	if len(tSubnet.FirstAutoIP) > 0 {
		startIp = tSubnet.FirstAutoIP
	} else {
		startIp = copyIp(networkIp)
		incrementIp(startIp)
	}
	if len(tSubnet.LastAutoIP) > 0 {
		stopIp = tSubnet.LastAutoIP
		incrementIp(stopIp)
	} else {
		stopIp = make(net.IP, len(networkIp))
		for index, value := range m.invertIP(tSubnet.IpMask) {
			stopIp[index] = networkIp[index] | value
		}
	}
	return &subnetType{
		subnet:  tSubnet,
		startIp: startIp,
		stopIp:  stopIp,
		nextIp:  copyIp(startIp),
	}
}

func (m *Manager) unmarkAllocatingIPs(ips []net.IP) {
	if len(ips) < 1 {
		return
	}
	m.mutex.Lock()
	defer m.mutex.Unlock()
	for _, ip := range ips {
		delete(m.allocatingIPs, ip.String())
	}
}
