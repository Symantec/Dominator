package configurator

import (
	"fmt"
	"net"
	"sort"
	"strings"

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

func findSubnet(subnets []*hyper_proto.Subnet,
	subnetId string) *hyper_proto.Subnet {
	for _, subnet := range subnets {
		if subnet.Id == subnetId {
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

func (netconf *NetworkConfig) addBridgeOnlyInterface(iface net.Interface,
	subnetId string) {
	netconf.bridgeOnlyInterfaces = append(netconf.bridgeOnlyInterfaces,
		bridgeOnlyInterfaceType{
			netInterface: iface,
			subnetId:     subnetId,
		})
}

func (netconf *NetworkConfig) addNormalInterface(iface net.Interface,
	ipAddr net.IP, subnet *hyper_proto.Subnet) {
	netconf.normalInterfaces = append(netconf.normalInterfaces,
		normalInterfaceType{
			netInterface: iface,
			ipAddr:       ipAddr,
			subnet:       subnet,
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
			if len(networkEntry.HostMacAddress) < 1 ||
				networkEntry.SubnetId == "" {
				continue
			}
			iface, ok := hwAddrToInterface[networkEntry.HostMacAddress.String()]
			if !ok {
				return nil, fmt.Errorf("MAC address: %s not found",
					networkEntry.HostMacAddress)
			}
			subnet := findSubnet(machineInfo.Subnets, networkEntry.SubnetId)
			if subnet == nil {
				return nil,
					fmt.Errorf("subnetId: %s not found", networkEntry.SubnetId)
			}
			if !subnet.Manage {
				return nil,
					fmt.Errorf("subnetId: %s is not managed", subnet.Id)
			}
			if len(subnet.Id) >= 10 {
				return nil,
					fmt.Errorf("subnetId: %s is over 9 characters", subnet.Id)
			}
			if strings.ContainsAny(subnet.Id, "/.") {
				return nil,
					fmt.Errorf("subnetId: %s contains '/' or '.'", subnet.Id)
			}
			usedSubnets[subnet] = struct{}{}
			netconf.addBridgeOnlyInterface(iface, networkEntry.SubnetId)
			delete(interfaces, iface.Name)
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
		netconf.addNormalInterface(iface, networkEntry.HostIpAddress, subnet)
		delete(interfaces, iface.Name)
		if subnet == preferredSubnet {
			netconf.DefaultSubnet = subnet
		} else if netconf.DefaultSubnet == nil {
			netconf.DefaultSubnet = subnet
		}
	}
	// All remaining interfaces which are marked as up will be used for VLAN
	// trunk. If there multiple interfaces, they will be bonded.
	for name, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 {
			delete(interfaces, name)
		} else {
			netconf.bondSlaves = append(netconf.bondSlaves, name)
			netconf.vlanRawDevice = name
		}
	}
	if len(interfaces) > 1 {
		netconf.vlanRawDevice = "bond0"
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
			entryName := fmt.Sprintf("%s.%d",
				netconf.vlanRawDevice, subnet.VlanId)
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
			if subnet.VlanId > 0 {
				netconf.bridges = append(netconf.bridges, subnet.VlanId)
			}
		}
	}
	return netconf, nil
}
