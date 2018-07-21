package util

import (
	"errors"
	"net"
	"os"
	"path/filepath"
)

const procNetVlan = "/proc/net/vlan"

func getMyIP() (net.IP, error) {
	var myIP net.IP
	mostOnesInMask := 0
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		if iface.Flags&net.FlagBroadcast == 0 {
			continue
		}
		if _, e := os.Stat(filepath.Join(procNetVlan, iface.Name)); e == nil {
			continue
		}
		interfaceAddrs, err := iface.Addrs()
		if err != nil {
			return nil, err
		}
		for _, addr := range interfaceAddrs {
			IP, IPNet, err := net.ParseCIDR(addr.String())
			if err != nil {
				return nil, err
			}
			if IP = IP.To4(); IP == nil {
				continue
			}
			if onesInMask, _ := IPNet.Mask.Size(); onesInMask > mostOnesInMask {
				myIP = IP
				mostOnesInMask = onesInMask
			}
		}
	}
	if myIP == nil {
		return nil, errors.New("no IP address found")
	}
	return myIP, nil
}
