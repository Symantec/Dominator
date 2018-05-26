package fsstorer

import (
	"errors"
	"net"
)

func (s *IpStorer) checkIpIsRegistered(addr net.IP) (bool, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if ip, err := netIpToIp(addr); err != nil {
		return false, err
	} else {
		_, ok := s.ipToHypervisor[ip]
		return ok, nil
	}
}

func netIpToIp(netIP net.IP) (IP, error) {
	switch len(netIP) {
	case 4:
	case 16:
		netIP = netIP.To4()
		if netIP == nil {
			return zeroIP, errors.New("bad IP")
		}
	default:
		return zeroIP, errors.New("bad IP")
	}
	var ip IP
	copy(ip[:], netIP)
	return ip, nil
}
