package main

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/lib/log"
	fm_proto "github.com/Symantec/Dominator/proto/fleetmanager"
	hyper_proto "github.com/Symantec/Dominator/proto/hypervisor"
	"github.com/d2g/dhcp4"
	"github.com/d2g/dhcp4client"
	"github.com/pin/tftp"
)

var (
	tftpFiles = []string{
		"config.json",
		"imagename",
		"imageserver",
		"storage-layout.json",
	}
	zeroIP = net.IP(make([]byte, 4))
)

func configureLocalNetwork(logger log.DebugLogger) (
	*fm_proto.GetMachineInfoResponse, map[string]net.Interface, error) {
	if err := run("ifconfig", "", logger, "lo", "up"); err != nil {
		return nil, nil, err
	}
	interfaces, err := listInterfaces(logger)
	if err != nil {
		return nil, nil, err
	}
	// Raise interfaces so that by the time the OS is installed link status
	// should be stable. This is how we discover connected interfaces.
	if err := raiseInterfaces(interfaces, logger); err != nil {
		return nil, nil, err
	}
	machineInfo, err := getConfiguration(interfaces, logger)
	if err != nil {
		return nil, nil, err
	}
	return machineInfo, interfaces, nil
}

func dhcpRequest(interfaces map[string]net.Interface,
	logger log.DebugLogger) (string, dhcp4.Packet, error) {
	clients := make(map[string]*dhcp4client.Client, len(interfaces))
	for name, iface := range interfaces {
		packetSocket, err := dhcp4client.NewPacketSock(iface.Index)
		if err != nil {
			return "", nil, err
		}
		defer packetSocket.Close()
		client, err := dhcp4client.New(
			dhcp4client.HardwareAddr(iface.HardwareAddr),
			dhcp4client.Connection(packetSocket),
			dhcp4client.Timeout(time.Second*5))
		if err != nil {
			return "", nil, err
		}
		defer client.Close()
		clients[name] = client
	}
	stopTime := time.Now().Add(time.Minute * 5)
	for ; time.Until(stopTime) > 0; time.Sleep(time.Second) {
		for name, client := range clients {
			if testCarrier(name) {
				logger.Debugf(1, "%s: DHCP attempt\n", name)
				if ok, packet, err := client.Request(); err != nil {
					logger.Debugf(1, "%s: DHCP failed: %s\n", name, err)
				} else if ok {
					return name, packet, nil
				}
			}
		}
	}
	return "", nil, errors.New("timed out waiting for DHCP")
}

func findInterfaceToConfigure(interfaces map[string]net.Interface,
	machineInfo fm_proto.GetMachineInfoResponse, logger log.DebugLogger) (
	net.Interface, net.IP, *hyper_proto.Subnet, error) {
	networkEntries := getNetworkEntries(machineInfo)
	hwAddrToInterface := make(map[string]net.Interface, len(interfaces))
	for _, iface := range interfaces {
		hwAddrToInterface[iface.HardwareAddr.String()] = iface
	}
	for _, networkEntry := range networkEntries {
		if len(networkEntry.HostIpAddress) < 1 {
			continue
		}
		iface, ok := hwAddrToInterface[networkEntry.HostMacAddress.String()]
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
		return iface, networkEntry.HostIpAddress, subnet, nil
	}
	return net.Interface{}, nil, nil,
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

func getConfiguration(interfaces map[string]net.Interface,
	logger log.DebugLogger) (*fm_proto.GetMachineInfoResponse, error) {
	var machineInfo fm_proto.GetMachineInfoResponse
	err := json.ReadFromFile(filepath.Join(*tftpDirectory, "config.json"),
		&machineInfo)
	if err == nil { // Configuration was injected.
		err := setupNetworkFromConfig(interfaces, machineInfo, logger)
		if err != nil {
			return nil, err
		}
		return &machineInfo, nil
	}
	if !os.IsNotExist(err) {
		return nil, err
	}
	if err := setupNetworkFromDhcp(interfaces, logger); err != nil {
		return nil, err
	}
	err = json.ReadFromFile(filepath.Join(*tftpDirectory, "config.json"),
		&machineInfo)
	return &machineInfo, nil
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

func listInterfaces(logger log.DebugLogger) (map[string]net.Interface, error) {
	interfaces := make(map[string]net.Interface)
	if allInterfaces, err := net.Interfaces(); err != nil {
		return nil, err
	} else {
		for _, iface := range allInterfaces {
			if iface.Flags&net.FlagBroadcast == 0 {
				logger.Debugf(2, "skipping non-EtherNet interface: %s\n",
					iface.Name)
			} else {
				logger.Debugf(1, "found EtherNet interface: %s\n", iface.Name)
				interfaces[iface.Name] = iface
			}
		}
	}
	return interfaces, nil
}

func loadTftpFiles(tftpServer net.IP, logger log.DebugLogger) error {
	client, err := tftp.NewClient(tftpServer.String() + ":69")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(*tftpDirectory, dirPerms); err != nil {
		return err
	}
	for _, name := range tftpFiles {
		logger.Debugf(1, "downloading: %s\n", name)
		if wt, err := client.Receive(name, "octet"); err != nil {
			return err
		} else {
			filename := filepath.Join(*tftpDirectory, name)
			if file, err := create(filename); err != nil {
				return err
			} else {
				defer file.Close()
				if _, err := wt.WriteTo(file); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func raiseInterfaces(interfaces map[string]net.Interface,
	logger log.DebugLogger) error {
	for name := range interfaces {
		if err := run("ifconfig", "", logger, name, "up"); err != nil {
			return err
		}
	}
	return nil
}

func setupNetwork(ifName string, ipAddr net.IP, subnet *hyper_proto.Subnet,
	logger log.DebugLogger) error {
	err := run("ifconfig", "", logger, ifName, ipAddr.String(), "netmask",
		subnet.IpMask.String(), "up")
	if err != nil {
		return err
	}
	err = run("route", "", logger, "add", "default", "gw",
		subnet.IpGateway.String())
	if err != nil {
		e := run("route", "", logger, "del", "default", "gw",
			subnet.IpGateway.String())
		if e != nil {
			return err
		}
		err = run("route", "", logger, "add", "default", "gw",
			subnet.IpGateway.String())
		if err != nil {
			return err
		}
	}
	if err := writeResolvConf("/etc/resolv.conf", subnet); err != nil {
		return err
	}
	return nil
}

func setupNetworkFromConfig(interfaces map[string]net.Interface,
	machineInfo fm_proto.GetMachineInfoResponse, logger log.DebugLogger) error {
	iface, ipAddr, subnet, err := findInterfaceToConfigure(interfaces,
		machineInfo, logger)
	if err != nil {
		return err
	}
	return setupNetwork(iface.Name, ipAddr, subnet, logger)
}

func setupNetworkFromDhcp(interfaces map[string]net.Interface,
	logger log.DebugLogger) error {
	ifName, packet, err := dhcpRequest(interfaces, logger)
	if err != nil {
		return err
	}
	ipAddr := packet.YIAddr()
	options := packet.ParseOptions()
	subnet := hyper_proto.Subnet{
		IpGateway: net.IP(options[dhcp4.OptionRouter]),
		IpMask:    net.IP(options[dhcp4.OptionSubnetMask]),
	}
	dnsServersBuffer := options[dhcp4.OptionDomainNameServer]
	for len(dnsServersBuffer) > 0 {
		if len(dnsServersBuffer) >= 4 {
			subnet.DomainNameServers = append(subnet.DomainNameServers,
				net.IP(dnsServersBuffer[:4]))
			dnsServersBuffer = dnsServersBuffer[4:]
		} else {
			return errors.New("truncated DNS server address")
		}
	}
	if err := setupNetwork(ifName, ipAddr, &subnet, logger); err != nil {
		return err
	}
	tftpServer := packet.SIAddr()
	if tftpServer.Equal(zeroIP) {
		tftpServer = net.IP(options[dhcp4.OptionTFTPServerName])
		if tftpServer.Equal(zeroIP) {
			return errors.New("no TFTP server given")
		}
		logger.Printf("tftpServer from OptionTFTPServerName: %s\n", tftpServer)
	} else {
		logger.Printf("tftpServer from SIAddr: %s\n", tftpServer)
	}
	return loadTftpFiles(tftpServer, logger)
}

func writeResolvConf(filename string, subnet *hyper_proto.Subnet) error {
	if file, err := create(filename); err != nil {
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
		if err := writer.Flush(); err != nil {
			return err
		}
	}
	return nil
}
