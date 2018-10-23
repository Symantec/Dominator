package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"

	imageclient "github.com/Symantec/Dominator/imageserver/client"
	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/image"
	libjson "github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
	fm_proto "github.com/Symantec/Dominator/proto/fleetmanager"
	hyper_proto "github.com/Symantec/Dominator/proto/hypervisor"
	installer_proto "github.com/Symantec/Dominator/proto/installer"
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

func getStorageLayout() (installer_proto.StorageLayout, error) {
	if *storageLayoutFilename == "" {
		return makeDefaultStorageLayout(), nil
	}
	var val installer_proto.StorageLayout
	if err := libjson.ReadFromFile(*storageLayoutFilename, &val); err != nil {
		return installer_proto.StorageLayout{}, err
	}
	return val, nil
}

func makeConfigFiles(info fm_proto.GetMachineInfoResponse, img *image.Image,
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
		filesMap["config.json"] = append(data, '\n')
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
	fmt.Fprintf(buffer, "%s:%d\n", *imageServerHostname, *imageServerPortNum)
	filesMap["objectserver"] = buffer.Bytes()
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
	if layout, err := getStorageLayout(); err != nil {
		return nil, err
	} else {
		var imageSize uint64
		if img == nil {
			imageSize = 1 << 29
		} else {
			imageSize = img.FileSystem.EstimateUsage(0)
		}
		for index := range layout.BootDriveLayout {
			if layout.BootDriveLayout[index].MountPoint != "/" {
				continue
			}
			layout.BootDriveLayout[index].MinimumFreeBytes += imageSize
			break
		}
		if data, err := json.MarshalIndent(layout, "", "    "); err != nil {
			return nil, err
		} else {
			filesMap["storage-layout.json"] = append(data, '\n')
		}
	}
	return filesMap, nil
}

func makeDefaultStorageLayout() installer_proto.StorageLayout {
	return installer_proto.StorageLayout{
		BootDriveLayout: []installer_proto.Partition{
			{
				MountPoint:       "/",
				MinimumFreeBytes: 256 << 20,
			},
			{
				MountPoint:       "/home",
				MinimumFreeBytes: 1 << 30,
			},
			{
				MountPoint:       "/var/log",
				MinimumFreeBytes: 256 << 20,
			},
		},
		ExtraMountPointsBasename: "/data/",
	}
}

func netbootHost(hostname string, logger log.DebugLogger) error {
	fmCR := srpc.NewClientResource("tcp",
		fmt.Sprintf("%s:%d", *fleetManagerHostname, *fleetManagerPortNum))
	defer fmCR.ScheduleClose()
	info, err := getInfoForMachine(fmCR, hostname)
	if err != nil {
		return err
	}
	imageName := info.Machine.Tags["RequiredImage"]
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
	defer hyperCR.ScheduleClose()
	var img *image.Image
	if imageName != "" {
		imageClient, err := srpc.DialHTTP("tcp", fmt.Sprintf("%s:%d",
			*imageServerHostname, *imageServerPortNum), 0)
		if err != nil {
			return err
		}
		defer imageClient.Close()
		img, err = imageclient.GetImage(imageClient, imageName)
		if err != nil {
			return err
		}
		if img == nil {
			return fmt.Errorf("image: %s does not exist", imageName)
		}
	}
	filesMap, err := makeConfigFiles(info, img, networkEntries)
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
