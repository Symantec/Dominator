package net

import (
	"net"
	"os"
	"path"
)

const sysClassNet = "/sys/class/net"

func listBridges() ([]net.Interface, error) {
	if file, err := os.Open(sysClassNet); err != nil {
		return nil, err
	} else {
		file.Close()
	}
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	var bridges []net.Interface
	for _, iface := range interfaces {
		pathname := path.Join(sysClassNet, iface.Name, "bridge")
		if _, err := os.Stat(pathname); err != nil {
			if !os.IsNotExist(err) {
				return nil, err
			}
		} else {
			bridges = append(bridges, iface)
		}
	}
	return bridges, nil
}
