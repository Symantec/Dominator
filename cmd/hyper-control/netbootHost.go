package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"

	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
	fm_proto "github.com/Symantec/Dominator/proto/fleetmanager"
	hyper_proto "github.com/Symantec/Dominator/proto/hypervisor"
)

type hostAddressType struct {
	address  hyper_proto.Address
	hostname string
}

type leaseType struct {
	address hostAddressType
	subnet  *hyper_proto.Subnet
}

func netbootHostSubcommand(args []string, logger log.DebugLogger) {
	err := netbootHost(args[0], logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error netbooting host: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
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

func getHostAddress(networkEntries []fm_proto.NetworkEntry) []hostAddressType {
	hostAddresses := make([]hostAddressType, 0, len(networkEntries))
	for _, networkEntry := range networkEntries {
		if len(networkEntry.HostIpAddress) > 0 &&
			len(networkEntry.HostMacAddress) > 0 {
			hostAddresses = append(hostAddresses, hostAddressType{
				address: hyper_proto.Address{
					IpAddress:  networkEntry.HostIpAddress,
					MacAddress: networkEntry.HostMacAddress.String(),
				},
				hostname: networkEntry.Hostname,
			})
		}
	}
	return hostAddresses
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

func makeConfigFiles(info fm_proto.GetMachineInfoResponse,
	networkEntries []fm_proto.NetworkEntry) (map[string][]byte, error) {
	filesMap := make(map[string][]byte, len(netbootFiles)+1)
	for tftpFilename, localFilename := range netbootFiles {
		if data, err := ioutil.ReadFile(localFilename); err != nil {
			return nil, err
		} else {
			filesMap[tftpFilename] = data
		}
	}
	if data, err := json.MarshalIndent(info, "", "    "); err != nil {
		return nil, err
	} else {
		filesMap["config.json"] = data
	}
	ifIndex := 0
	buffer := new(bytes.Buffer)
	var primarySubnet *hyper_proto.Subnet
	for _, networkEntry := range networkEntries {
		subnet := findMatchingSubnet(info.Subnets, networkEntry.HostIpAddress)
		if subnet == nil {
			continue
		}
		if ifIndex == 0 {
			primarySubnet = subnet
		}
		fmt.Fprintf(buffer, "ip%d=%s\n", ifIndex, networkEntry.HostIpAddress)
		fmt.Fprintf(buffer, "gateway%d=%s\n", ifIndex, subnet.IpGateway)
		fmt.Fprintf(buffer, "mask%d=%s\n", ifIndex, subnet.IpMask)
		ifIndex++
	}
	if primarySubnet == nil {
		return nil, errors.New("no primary subnet found")
	}
	filesMap["network-interfaces"] = buffer.Bytes()
	buffer = new(bytes.Buffer)
	if primarySubnet.DomainName != "" {
		fmt.Fprintf(buffer, "domain %s\n", primarySubnet.DomainName)
		fmt.Fprintf(buffer, "search %s\n", primarySubnet.DomainName)
		fmt.Fprintln(buffer)
	}
	for _, nameserver := range primarySubnet.DomainNameServers {
		fmt.Fprintf(buffer, "nameserver %s\n", nameserver)
	}
	filesMap["resolv.conf"] = buffer.Bytes()
	return filesMap, nil
}

func netbootHost(hostname string, logger log.DebugLogger) error {
	fmCR := srpc.NewClientResource("tcp",
		fmt.Sprintf("%s:%d", *fleetManagerHostname, *fleetManagerPortNum))
	defer fmCR.ScheduleClose()
	info, err := getInfoForMachine(fmCR, hostname)
	if err != nil {
		return err
	}
	subnets := make([]*hyper_proto.Subnet, 0, len(info.Subnets))
	for _, subnet := range info.Subnets {
		if subnet.VlanId == 0 {
			subnets = append(subnets, subnet)
		}
	}
	if len(subnets) < 1 {
		return errors.New("no non-VLAN subnets known")
	}
	networkEntries := getNetworkEntries(info)
	hostAddresses := getHostAddress(networkEntries)
	if len(hostAddresses) < 1 {
		return errors.New("no IP and MAC addresses known for host")
	}
	leases := make([]leaseType, 0, len(hostAddresses))
	for _, address := range hostAddresses {
		subnet := findMatchingSubnet(subnets, address.address.IpAddress)
		if subnet != nil {
			leases = append(leases, leaseType{address: address, subnet: subnet})
		}
	}
	if len(leases) < 1 {
		return errors.New("no IP and MAC addresses matching a subnet")
	}
	hypervisorAddresses, err := listGoodHypervisorsInLocation(fmCR,
		info.Location)
	if err != nil {
		return err
	}
	if len(hypervisorAddresses) < 1 {
		return errors.New("no nearby Hypervisors available")
	}
	logger.Debugf(0, "Selected %s as boot server on subnet: %s\n",
		hypervisorAddresses[0], leases[0].subnet.Id)
	hyperCR := srpc.NewClientResource("tcp", hypervisorAddresses[0])
	defer fmCR.ScheduleClose()
	filesMap, err := makeConfigFiles(info, networkEntries)
	if err != nil {
		return err
	}
	request := hyper_proto.NetbootMachineRequest{
		Address:                      leases[0].address.address,
		Files:                        filesMap,
		FilesExpiration:              *netbootFilesTimeout,
		Hostname:                     hostname,
		NumAcknowledgementsToWaitFor: *numAcknowledgementsToWaitFor,
		OfferExpiration:              *offerTimeout,
		WaitTimeout:                  *netbootTimeout,
	}
	var reply hyper_proto.NetbootMachineResponse
	client, err := hyperCR.GetHTTP(nil, 0)
	if err != nil {
		return err
	}
	defer client.Put()
	err = client.RequestReply("Hypervisor.NetbootMachine", request, &reply)
	if err != nil {
		return err
	}
	return errors.New(reply.Error)
}
