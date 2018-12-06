package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/Symantec/Dominator/lib/log"
	fm_proto "github.com/Symantec/Dominator/proto/fleetmanager"
	hyper_proto "github.com/Symantec/Dominator/proto/hypervisor"
)

type bondedInterfaceType struct {
	name   string // Physical interface name.
	ipAddr net.IP
	subnet *hyper_proto.Subnet
}

type normalInterfaceType struct {
	name   string // Physical interface name.
	ipAddr net.IP
	subnet *hyper_proto.Subnet
}

type networkConfig struct {
	bondedInterfaces []bondedInterfaceType
	bridges          []uint
	normalInterfaces []normalInterfaceType
	bondSlaves       []string // New interface name.
}

func addMapping(mappings map[string]string, name string) error {
	filename := fmt.Sprintf("/sys/class/net/%s/device", name)
	if symlink, err := os.Readlink(filename); err != nil {
		return err
	} else {
		mappings[name] = filepath.Base(symlink)
		return nil
	}
}

func configureNetwork(machineInfo fm_proto.GetMachineInfoResponse,
	interfaces map[string]net.Interface, logger log.DebugLogger) error {
	hostname := strings.Split(machineInfo.Machine.Hostname, ".")[0]
	err := ioutil.WriteFile(filepath.Join(*mountPoint, "etc", "hostname"),
		[]byte(hostname+"\n"), filePerms)
	if err != nil {
		return err
	}
	netconf := &networkConfig{}
	networkEntries := getNetworkEntries(machineInfo)
	mappings := make(map[string]string)
	for name := range interfaces {
		if err := addMapping(mappings, name); err != nil {
			return err
		}
	}
	connectedInterfaces := getConnectedInterfaces(interfaces, logger)
	hwAddrToInterface := make(map[string]net.Interface, len(interfaces))
	for _, iface := range interfaces {
		hwAddrToInterface[iface.HardwareAddr.String()] = iface
	}
	var defaultSubnet *hyper_proto.Subnet
	preferredSubnet := findMatchingSubnet(machineInfo.Subnets,
		machineInfo.Machine.HostIpAddress)
	// First process network entries with normal interfaces.
	var bondedNetworkEntries []fm_proto.NetworkEntry
	normalInterfaceIndex := 0
	usedSubnets := make(map[*hyper_proto.Subnet]struct{})
	for _, networkEntry := range networkEntries {
		if len(networkEntry.HostIpAddress) < 1 {
			continue
		}
		if len(networkEntry.HostMacAddress) < 1 {
			bondedNetworkEntries = append(bondedNetworkEntries, networkEntry)
			continue
		}
		iface, ok := hwAddrToInterface[networkEntry.HostMacAddress.String()]
		if !ok {
			logger.Printf("MAC address: %s not found\n",
				networkEntry.HostMacAddress)
			continue
		}
		subnet := findMatchingSubnet(machineInfo.Subnets,
			networkEntry.HostIpAddress)
		if subnet == nil {
			logger.Printf("no matching subnet for ip=%s\n",
				networkEntry.HostIpAddress)
			continue
		}
		usedSubnets[subnet] = struct{}{}
		normalInterfaceIndex++
		netconf.addNormalInterface(iface.Name, networkEntry.HostIpAddress,
			subnet)
		delete(connectedInterfaces, iface.Name)
		if subnet == preferredSubnet {
			defaultSubnet = subnet
		} else if defaultSubnet == nil {
			defaultSubnet = subnet
		}
	}
	for name := range connectedInterfaces {
		netconf.bondSlaves = append(netconf.bondSlaves, name)
	}
	if len(connectedInterfaces) > 0 {
		for _, networkEntry := range bondedNetworkEntries {
			subnet := findMatchingSubnet(machineInfo.Subnets,
				networkEntry.HostIpAddress)
			if subnet == nil {
				logger.Printf("no matching subnet for ip=%s\n",
					networkEntry.HostIpAddress)
				continue
			}
			usedSubnets[subnet] = struct{}{}
			entryName := fmt.Sprintf("bond0.%d", subnet.VlanId)
			netconf.addBondedInterface(entryName, networkEntry.HostIpAddress,
				subnet)
			if subnet == preferredSubnet {
				defaultSubnet = subnet
			} else if defaultSubnet == nil {
				defaultSubnet = subnet
			}
		}
		for _, subnet := range machineInfo.Subnets {
			if _, ok := usedSubnets[subnet]; ok {
				continue
			}
			netconf.bridges = append(netconf.bridges, subnet.VlanId)
		}
	}
	if err := netconf.write(defaultSubnet.IpGateway); err != nil {
		return err
	}
	if err := writeMappings(mappings); err != nil {
		return err
	}
	err = writeResolvConf(filepath.Join(*mountPoint, "etc", "resolv.conf"),
		defaultSubnet)
	if err != nil {
		return err
	}
	return nil
}

func getConnectedInterfaces(interfaces map[string]net.Interface,
	logger log.DebugLogger) map[string]net.Interface {
	connectedInterfaces := make(map[string]net.Interface)
	for name, iface := range interfaces {
		if testCarrier(name) {
			connectedInterfaces[name] = iface
			logger.Debugf(1, "%s is connected\n", name)
			continue
		}
		run("ifconfig", "", logger, name, "down")
	}
	return connectedInterfaces
}

func testCarrier(name string) bool {
	filename := fmt.Sprintf("/sys/class/net/%s/carrier", name)
	if data, err := ioutil.ReadFile(filename); err == nil {
		if len(data) > 0 && data[0] == '1' {
			return true
		}
	}
	return false
}

func (netconf *networkConfig) addBondedInterface(name string, ipAddr net.IP,
	subnet *hyper_proto.Subnet) {
	netconf.bondedInterfaces = append(netconf.bondedInterfaces,
		bondedInterfaceType{
			name:   name,
			ipAddr: ipAddr,
			subnet: subnet,
		})
}

func (netconf *networkConfig) addNormalInterface(name string, ipAddr net.IP,
	subnet *hyper_proto.Subnet) {
	netconf.normalInterfaces = append(netconf.normalInterfaces,
		normalInterfaceType{
			name:   name,
			ipAddr: ipAddr,
			subnet: subnet,
		})
}

func (netconf *networkConfig) write(ipGateway net.IP) error {
	return netconf.writeDebian(ipGateway)
}

func (netconf *networkConfig) writeDebian(ipGateway net.IP) error {
	filename := filepath.Join(*mountPoint, "etc", "network", "interfaces")
	if file, err := create(filename); err != nil {
		return err
	} else {
		defer file.Close()
		writer := bufio.NewWriter(file)
		defer writer.Flush()
		fmt.Fprintln(writer,
			"# /etc/network/interfaces -- created by SmallStack installer")
		fmt.Fprintln(writer)
		fmt.Fprintln(writer, "auto lo")
		fmt.Fprintln(writer, "iface lo inet loopback")
		for _, iface := range netconf.normalInterfaces {
			fmt.Fprintln(writer)
			fmt.Fprintf(writer, "auto %s\n", iface.name)
			fmt.Fprintf(writer, "iface %s inet static\n", iface.name)
			fmt.Fprintf(writer, "\taddress %s\n", iface.ipAddr)
			fmt.Fprintf(writer, "\tnetmask %s\n", iface.subnet.IpMask)
			if iface.subnet.IpGateway.Equal(ipGateway) {
				fmt.Fprintf(writer, "\tgateway %s\n", iface.subnet.IpGateway)
			}
		}
		if len(netconf.bondSlaves) > 0 {
			fmt.Fprintln(writer)
			fmt.Fprintln(writer, "auto bond0")
			fmt.Fprintln(writer, "iface bond0 inet manual")
			fmt.Fprintln(writer, "\tup ip link set bond0 mtu 9000")
			fmt.Fprintln(writer, "\tbond-mode 802.3ad")
			fmt.Fprintln(writer, "\tbond-xmit_hash_policy 1")
			fmt.Fprint(writer, "\tslaves")
			for _, name := range netconf.bondSlaves {
				fmt.Fprint(writer, " ", name)
			}
			fmt.Fprintln(writer)
		}
		for _, iface := range netconf.bondedInterfaces {
			fmt.Fprintln(writer)
			fmt.Fprintf(writer, "auto %s\n", iface.name)
			fmt.Fprintf(writer, "iface %s inet static\n", iface.name)
			fmt.Fprintln(writer, "\tvlan-raw-device bond0")
			fmt.Fprintf(writer, "\taddress %s\n", iface.ipAddr)
			fmt.Fprintf(writer, "\tnetmask %s\n", iface.subnet.IpMask)
			if iface.subnet.IpGateway.Equal(ipGateway) {
				fmt.Fprintf(writer, "\tgateway %s\n", iface.subnet.IpGateway)
			}
		}
		for _, vlanId := range netconf.bridges {
			fmt.Fprintln(writer)
			fmt.Fprintf(writer, "auto bond0.%d\n", vlanId)
			fmt.Fprintf(writer, "iface bond0.%d inet manual\n", vlanId)
			fmt.Fprintln(writer, "\tvlan-raw-device bond0")
			fmt.Fprintln(writer)
			fmt.Fprintf(writer, "auto br%d\n", vlanId)
			fmt.Fprintf(writer, "iface br%d inet manual\n", vlanId)
			fmt.Fprintf(writer, "\tbridge_ports bond0.%d\n", vlanId)
		}
		return nil
	}
}

func writeMappings(mappings map[string]string) error {
	filename := filepath.Join(*mountPoint,
		"etc", "udev", "rules.d", "70-persistent-net.rules")
	if file, err := create(filename); err != nil {
		return err
	} else {
		defer file.Close()
		writer := bufio.NewWriter(file)
		defer writer.Flush()
		for name, kernelId := range mappings {
			fmt.Fprintf(writer,
				`SUBSYSTEM=="net", ACTION=="add", DRIVERS=="?*", ATTR{type}=="1", KERNELS=="%s", NAME="%s"`,
				kernelId, name)
			fmt.Fprintln(writer)
		}
		return writer.Flush()
	}
}
