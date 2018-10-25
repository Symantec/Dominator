package main

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/lib/log"
	fm_proto "github.com/Symantec/Dominator/proto/fleetmanager"
	hyper_proto "github.com/Symantec/Dominator/proto/hypervisor"
)

func configureNetwork(logger log.DebugLogger) (
	*fm_proto.GetMachineInfoResponse, error) {
	if err := run("ifconfig", "", logger, "lo", "up"); err != nil {
		return nil, err
	}
	interfaces, err := listInterfaces(logger)
	if err != nil {
		return nil, err
	}
	machineInfo, err := getConfiguration(interfaces, logger)
	if err != nil {
		return nil, err
	}
	return machineInfo, nil
}

func findInterfaceToConfigure(interfaces []net.Interface,
	machineInfo fm_proto.GetMachineInfoResponse,
	logger log.DebugLogger) (string, net.IP, *hyper_proto.Subnet, error) {
	networkEntries := getNetworkEntries(machineInfo)
	interfaceTable := make(map[string]net.Interface, len(interfaces))
	for _, iface := range interfaces {
		interfaceTable[iface.HardwareAddr.String()] = iface
	}
	for _, networkEntry := range networkEntries {
		if len(networkEntry.HostIpAddress) < 1 {
			continue
		}
		iface, ok := interfaceTable[networkEntry.HostMacAddress.String()]
		if !ok {
			continue
		}
		subnet := findMatchingSubnet(machineInfo.Subnets,
			networkEntry.HostIpAddress)
		if subnet == nil {
			logger.Printf("no matching subnet for ip=%s\n",
				networkEntry.HostIpAddress)
			continue
		}
		return iface.Name, networkEntry.HostIpAddress, subnet, nil
	}
	if err := raiseInterfaces(interfaces, logger); err != nil {
		return "", nil, nil, err
	}
	//if err := lowerInterfaces(interfaces, name, logger); err != nil {
	//	return err
	//}
	return "", nil, nil,
		errors.New("no network interfaces match injected configuration")
}

func findMatchingSubnet(subnets []*hyper_proto.Subnet,
	ipAddr net.IP) *hyper_proto.Subnet {
	for _, subnet := range subnets {
		subnetMask := net.IPMask(subnet.IpMask)
		subnetAddr := subnet.IpGateway.Mask(subnetMask)
		if ipAddr.Mask(subnetMask).Equal(subnetAddr) {
			return subnet
		}
	}
	return nil
}

func getConfiguration(interfaces []net.Interface,
	logger log.DebugLogger) (*fm_proto.GetMachineInfoResponse, error) {
	var machineInfo fm_proto.GetMachineInfoResponse
	err := json.ReadFromFile(filepath.Join(*tftpDirectory, "config.json"),
		&machineInfo)
	if err == nil { // Configuration was injected.
		if err := setupNetwork(interfaces, machineInfo, logger); err != nil {
			return nil, err
		}
		return &machineInfo, nil
	}
	if !os.IsNotExist(err) {
		return nil, err
	}
	return nil, errors.New("DHCP/TFTP not implemented yet")
}

func getNetworkEntries(
	info fm_proto.GetMachineInfoResponse) []fm_proto.NetworkEntry {
	networkEntries := make([]fm_proto.NetworkEntry, 1,
		len(info.Machine.SecondaryNetworkEntries)+1)
	networkEntries[0] = info.Machine.NetworkEntry
	for _, networkEntry := range info.Machine.SecondaryNetworkEntries {
		networkEntries = append(networkEntries, networkEntry)
	}
	return networkEntries
}

func listInterfaces(logger log.DebugLogger) ([]net.Interface, error) {
	var interfaces []net.Interface
	if allInterfaces, err := net.Interfaces(); err != nil {
		return nil, err
	} else {
		for _, iface := range allInterfaces {
			if iface.Flags&net.FlagBroadcast == 0 {
				logger.Debugf(2, "skipping non-EtherNet interface: %s\n",
					iface.Name)
			} else {
				logger.Debugf(1, "found EtherNet interface: %s\n", iface.Name)
				interfaces = append(interfaces, iface)
			}
		}
	}
	return interfaces, nil
}

func lowerInterfaces(interfaces []net.Interface, keepUp string,
	logger log.DebugLogger) error {
	for _, iface := range interfaces {
		if iface.Name == keepUp {
			continue
		}
		if err := run("ifconfig", "", logger, iface.Name, "down"); err != nil {
			return err
		}
	}
	return nil
}

func raiseInterfaces(interfaces []net.Interface, logger log.DebugLogger) error {
	for _, iface := range interfaces {
		if err := run("ifconfig", "", logger, iface.Name, "up"); err != nil {
			return err
		}
	}
	return nil
}

func setupNetwork(interfaces []net.Interface,
	machineInfo fm_proto.GetMachineInfoResponse, logger log.DebugLogger) error {
	name, ipAddr, subnet, err := findInterfaceToConfigure(interfaces,
		machineInfo, logger)
	if err != nil {
		return err
	}
	err = run("ifconfig", "", logger, name, ipAddr.String(), "netmask",
		subnet.IpMask.String(), "up")
	if err != nil {
		return err
	}
	err = run("route", "", logger, "add", "default", "gw",
		subnet.IpGateway.String())
	if err != nil {
		return err
	}
	if file, err := create("/etc/resolv.conf"); err != nil {
		return err
	} else {
		defer file.Close()
		writer := bufio.NewWriter(file)
		defer writer.Flush()
		if subnet.DomainName != "" {
			fmt.Fprintf(writer, "domain %s\n", subnet.DomainName)
			fmt.Fprintf(writer, "search %s\n", subnet.DomainName)
			fmt.Fprintln(writer)
		}
		for _, nameserver := range subnet.DomainNameServers {
			fmt.Fprintf(writer, "nameserver %s\n", nameserver)
		}
	}
	return nil
}
