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
