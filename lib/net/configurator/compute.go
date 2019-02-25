package configurator

import (
	"fmt"
	"net"
	"sort"

	"github.com/Symantec/Dominator/lib/log"
	fm_proto "github.com/Symantec/Dominator/proto/fleetmanager"
	hyper_proto "github.com/Symantec/Dominator/proto/hypervisor"
)

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

func (netconf *NetworkConfig) addBondedInterface(name string, ipAddr net.IP,
	subnet *hyper_proto.Subnet) {
	netconf.bondedInterfaces = append(netconf.bondedInterfaces,
		bondedInterfaceType{
			name:   name,
			ipAddr: ipAddr,
			subnet: subnet,
		})
}

func (netconf *NetworkConfig) addNormalInterface(name string, ipAddr net.IP,
	subnet *hyper_proto.Subnet) {
	netconf.normalInterfaces = append(netconf.normalInterfaces,
		normalInterfaceType{
			name:   name,
			ipAddr: ipAddr,
			subnet: subnet,
		})
}

func compute(machineInfo fm_proto.GetMachineInfoResponse,
	_interfaces map[string]net.Interface,
	logger log.DebugLogger) (*NetworkConfig, error) {
	netconf := &NetworkConfig{}
	networkEntries := getNetworkEntries(machineInfo)
	interfaces := make(map[string]net.Interface, len(_interfaces))
	for name, iface := range _interfaces {
		interfaces[name] = iface
	}
	hwAddrToInterface := make(map[string]net.Interface, len(interfaces))
	for _, iface := range interfaces {
		hwAddrToInterface[iface.HardwareAddr.String()] = iface
	}
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
		delete(interfaces, iface.Name)
		if subnet == preferredSubnet {
			netconf.DefaultSubnet = subnet
		} else if netconf.DefaultSubnet == nil {
			netconf.DefaultSubnet = subnet
		}
	}
	for name := range interfaces {
		netconf.bondSlaves = append(netconf.bondSlaves, name)
	}
	sort.Strings(netconf.bondSlaves)
	if len(interfaces) > 0 {
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
				netconf.DefaultSubnet = subnet
			} else if netconf.DefaultSubnet == nil {
				netconf.DefaultSubnet = subnet
			}
		}
		for _, subnet := range machineInfo.Subnets {
			if _, ok := usedSubnets[subnet]; ok {
				continue
			}
			netconf.bridges = append(netconf.bridges, subnet.VlanId)
		}
	}
	return netconf, nil
}
