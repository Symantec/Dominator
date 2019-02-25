package net

import (
	"net"
	"os"
	"path/filepath"

	"github.com/Symantec/Dominator/lib/log"
)

const sysClassNet = "/sys/class/net"

func listBroadcastInterfaces(interfaceType uint, logger log.DebugLogger) (
	[]net.Interface, map[string]net.Interface, error) {
	interfaceList := make([]net.Interface, 0)
	interfaceMap := make(map[string]net.Interface)
	if allInterfaces, err := net.Interfaces(); err != nil {
		return nil, nil, err
	} else {
		for _, iface := range allInterfaces {
			if iface.Flags&net.FlagBroadcast == 0 {
				logger.Debugf(2, "skipping non-broadcast interface: %s\n",
					iface.Name)
			} else if includeType(iface, interfaceType, logger) {
				logger.Debugf(1, "found broadcast interface: %s\n", iface.Name)
				interfaceList = append(interfaceList, iface)
				interfaceMap[iface.Name] = iface
			}
		}
	}
	return interfaceList, interfaceMap, nil
}

func includeType(iface net.Interface, interfaceType uint,
	logger log.DebugLogger) bool {
	pathname := filepath.Join(sysClassNet, iface.Name, "bonding")
	if _, err := os.Stat(pathname); err == nil {
		if interfaceType&InterfaceTypeBonding == 0 {
			logger.Debugf(2, "skipping bonding interface: %s\n", iface.Name)
			return false
		} else {
			return true
		}
	}
	pathname = filepath.Join(sysClassNet, iface.Name, "bridge")
	if _, err := os.Stat(pathname); err == nil {
		if interfaceType&InterfaceTypeBridge == 0 {
			logger.Debugf(2, "skipping bridge interface: %s\n", iface.Name)
			return false
		} else {
			return true
		}
	}
	pathname = filepath.Join(sysClassNet, iface.Name, "device")
	if _, err := os.Stat(pathname); err == nil {
		if interfaceType&InterfaceTypeEtherNet == 0 {
			logger.Debugf(2, "skipping EtherNet interface: %s\n", iface.Name)
			return false
		} else {
			return true
		}
	}
	pathname = filepath.Join(procNetVlan, iface.Name)
	if _, err := os.Stat(pathname); err == nil {
		if interfaceType&InterfaceTypeVlan == 0 {
			logger.Debugf(2, "skipping Vlan interface: %s\n", iface.Name)
			return false
		} else {
			return true
		}
	}
	pathname = filepath.Join(sysClassNet, iface.Name, "tun_flags")
	if _, err := os.Stat(pathname); err == nil {
		if interfaceType&InterfaceTypeTunTap == 0 {
			logger.Debugf(2, "skipping TUN/TAP interface: %s\n", iface.Name)
			return false
		} else {
			return true
		}
	}
	logger.Debugf(1, "skipping unknown interface: %s\n", iface.Name)
	return false
}
