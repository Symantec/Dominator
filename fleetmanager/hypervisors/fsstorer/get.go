package fsstorer

import (
	"net"
)

func (s *Storer) getHypervisorForIp(addr net.IP) (net.IP, error) {
	if ip, err := netIpToIp(addr); err != nil {
		return nil, err
	} else {
		s.mutex.RLock()
		hypervisor, ok := s.ipToHypervisor[ip]
		s.mutex.RUnlock()
		if !ok {
			return nil, nil
		}
		return hypervisor[:], nil
	}
}

func (s *Storer) getIPsForHypervisor(hypervisor net.IP) ([]net.IP, error) {
	if hypervisorIP, err := netIpToIp(hypervisor); err != nil {
		return nil, err
	} else {
		s.mutex.RLock()
		ipList, ok := s.hypervisorToIPs[hypervisorIP]
		s.mutex.RUnlock()
		if !ok {
			return nil, nil
		}
		netIpList := make([]net.IP, 0, len(ipList))
		for _, ip := range ipList {
			ip := ip
			netIpList = append(netIpList, net.IP(ip[:]))
		}
		return netIpList, nil
	}
}
