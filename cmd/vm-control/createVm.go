package main

import (
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"time"

	hyperclient "github.com/Cloud-Foundations/Dominator/hypervisor/client"
	"github.com/Cloud-Foundations/Dominator/lib/format"
	"github.com/Cloud-Foundations/Dominator/lib/images/virtualbox"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/lib/tags"
	fm_proto "github.com/Cloud-Foundations/Dominator/proto/fleetmanager"
	hyper_proto "github.com/Cloud-Foundations/Dominator/proto/hypervisor"
)

type wrappedReadCloser struct {
	real io.Closer
	wrap io.Reader
}

func init() {
	rand.Seed(time.Now().Unix() + time.Now().UnixNano())
}

func createVmSubcommand(args []string, logger log.DebugLogger) error {
	if err := createVm(logger); err != nil {
		return fmt.Errorf("Error creating VM: %s", err)
	}
	return nil
}

func callCreateVm(client *srpc.Client, request hyper_proto.CreateVmRequest,
	reply *hyper_proto.CreateVmResponse, imageReader, userDataReader io.Reader,
	imageSize, userDataSize int64, logger log.DebugLogger) error {
	conn, err := client.Call("Hypervisor.CreateVm")
	if err != nil {
		return fmt.Errorf("error calling Hypervisor.CreateVm: %s", err)
	}
	defer conn.Close()
	if err := conn.Encode(request); err != nil {
		return fmt.Errorf("error encoding request: %s", err)
	}
	// Stream any required data.
	if imageReader != nil {
		logger.Debugln(0, "uploading image")
		startTime := time.Now()
		if nCopied, err := io.CopyN(conn, imageReader, imageSize); err != nil {
			return fmt.Errorf("error uploading image: %s got %d of %d bytes",
				err, nCopied, imageSize)
		} else {
			duration := time.Since(startTime)
			speed := uint64(float64(nCopied) / duration.Seconds())
			logger.Debugf(0, "uploaded image in %s (%s/s)\n",
				format.Duration(duration), format.FormatBytes(speed))
		}
	}
	if userDataReader != nil {
		logger.Debugln(0, "uploading user data")
		nCopied, err := io.CopyN(conn, userDataReader, userDataSize)
		if err != nil {
			return fmt.Errorf(
				"error uploading user data: %s got %d of %d bytes",
				err, nCopied, userDataSize)
		}
	}
	response, err := processCreateVmResponses(conn, logger)
	*reply = response
	return err
}

func createVm(logger log.DebugLogger) error {
	if *vmHostname == "" {
		if name := vmTags["Name"]; name == "" {
			return errors.New("no hostname specified")
		} else {
			*vmHostname = name
		}
	} else {
		if name := vmTags["Name"]; name == "" {
			if vmTags == nil {
				vmTags = make(tags.Tags)
			}
			vmTags["Name"] = *vmHostname
		}
	}
	if hypervisor, err := getHypervisorAddress(); err != nil {
		return err
	} else {
		logger.Debugf(0, "creating VM on %s\n", hypervisor)
		return createVmOnHypervisor(hypervisor, logger)
	}
}

func createVmInfoFromFlags() hyper_proto.VmInfo {
	return hyper_proto.VmInfo{
		ConsoleType:        consoleType,
		DestroyProtection:  *destroyProtection,
		DisableVirtIO:      *disableVirtIO,
		Hostname:           *vmHostname,
		MemoryInMiB:        uint64(memory >> 20),
		MilliCPUs:          *milliCPUs,
		OwnerGroups:        ownerGroups,
		OwnerUsers:         ownerUsers,
		Tags:               vmTags,
		SecondarySubnetIDs: secondarySubnetIDs,
		SubnetId:           *subnetId,
	}
}

func createVmOnHypervisor(hypervisor string, logger log.DebugLogger) error {
	request := hyper_proto.CreateVmRequest{
		DhcpTimeout:      *dhcpTimeout,
		EnableNetboot:    *enableNetboot,
		MinimumFreeBytes: uint64(minFreeBytes),
		RoundupPower:     *roundupPower,
		VmInfo:           createVmInfoFromFlags(),
	}
	if request.VmInfo.MemoryInMiB < 1 {
		request.VmInfo.MemoryInMiB = 1024
	}
	if request.VmInfo.MilliCPUs < 1 {
		request.VmInfo.MilliCPUs = 250
	}
	if len(requestIPs) > 0 && requestIPs[0] != "" {
		ipAddr := net.ParseIP(requestIPs[0])
		if ipAddr == nil {
			return fmt.Errorf("invalid IP address: %s", requestIPs[0])
		}
		request.Address.IpAddress = ipAddr
	}
	if len(requestIPs) > 1 && len(secondarySubnetIDs) > 0 {
		request.SecondaryAddresses = make([]hyper_proto.Address,
			len(secondarySubnetIDs))
		for index, addr := range requestIPs[1:] {
			if addr == "" {
				continue
			}
			ipAddr := net.ParseIP(addr)
			if ipAddr == nil {
				return fmt.Errorf("invalid IP address: %s", requestIPs[0])
			}
			request.SecondaryAddresses[index] = hyper_proto.Address{
				IpAddress: ipAddr}
		}
	}
	for _, size := range secondaryVolumeSizes {
		request.SecondaryVolumes = append(request.SecondaryVolumes,
			hyper_proto.Volume{Size: uint64(size)})
	}
	var imageReader, userDataReader io.Reader
	if *imageName != "" {
		request.ImageName = *imageName
		request.ImageTimeout = *imageTimeout
		request.SkipBootloader = *skipBootloader
	} else if *imageURL != "" {
		request.ImageURL = *imageURL
	} else if *imageFile != "" {
		file, size, err := getReader(*imageFile)
		if err != nil {
			return err
		} else {
			defer file.Close()
			request.ImageDataSize = uint64(size)
			imageReader = file
		}
	} else {
		return errors.New("no image specified")
	}
	if *userDataFile != "" {
		file, size, err := getReader(*userDataFile)
		if err != nil {
			return err
		} else {
			defer file.Close()
			request.UserDataSize = uint64(size)
			userDataReader = file
		}
	}
	client, err := dialHypervisor(hypervisor)
	if err != nil {
		return err
	}
	defer client.Close()
	var reply hyper_proto.CreateVmResponse
	err = callCreateVm(client, request, &reply, imageReader, userDataReader,
		int64(request.ImageDataSize), int64(request.UserDataSize), logger)
	if err != nil {
		return err
	}
	if err := hyperclient.AcknowledgeVm(client, reply.IpAddress); err != nil {
		return fmt.Errorf("error acknowledging VM: %s", err)
	}
	fmt.Println(reply.IpAddress)
	if reply.DhcpTimedOut {
		return errors.New("DHCP ACK timed out")
	}
	if *dhcpTimeout > 0 {
		logger.Debugln(0, "Received DHCP ACK")
	}
	return maybeWatchVm(client, hypervisor, reply.IpAddress, logger)
}

func getHypervisorAddress() (string, error) {
	if *hypervisorHostname != "" {
		return fmt.Sprintf("%s:%d", *hypervisorHostname, *hypervisorPortNum),
			nil
	}
	client, err := dialFleetManager(fmt.Sprintf("%s:%d",
		*fleetManagerHostname, *fleetManagerPortNum))
	if err != nil {
		return "", err
	}
	defer client.Close()
	if *adjacentVM != "" {
		if adjacentVmIpAddr, err := lookupIP(*adjacentVM); err != nil {
			return "", err
		} else {
			return findHypervisorClient(client, adjacentVmIpAddr)
		}
	}
	request := fm_proto.ListHypervisorsInLocationRequest{
		Location: *location,
		SubnetId: *subnetId,
	}
	var reply fm_proto.ListHypervisorsInLocationResponse
	err = client.RequestReply("FleetManager.ListHypervisorsInLocation",
		request, &reply)
	if err != nil {
		return "", err
	}
	if reply.Error != "" {
		return "", errors.New(reply.Error)
	}
	if numHyper := len(reply.HypervisorAddresses); numHyper < 1 {
		return "", errors.New("no active Hypervisors in location")
	} else if numHyper < 2 {
		return reply.HypervisorAddresses[0], nil
	} else {
		return reply.HypervisorAddresses[rand.Intn(numHyper-1)], nil
	}
}

func getReader(filename string) (io.ReadCloser, int64, error) {
	if file, err := os.Open(filename); err != nil {
		return nil, -1, err
	} else if filepath.Ext(filename) == ".vdi" {
		vdi, err := virtualbox.NewReader(file)
		if err != nil {
			file.Close()
			return nil, -1, err
		}
		return &wrappedReadCloser{real: file, wrap: vdi}, int64(vdi.Size), nil
	} else {
		fi, err := file.Stat()
		if err != nil {
			file.Close()
			return nil, -1, err
		}
		return file, fi.Size(), nil
	}
}

func processCreateVmResponses(conn *srpc.Conn,
	logger log.DebugLogger) (hyper_proto.CreateVmResponse, error) {
	var zeroResponse hyper_proto.CreateVmResponse
	if err := conn.Flush(); err != nil {
		return zeroResponse, fmt.Errorf("error flushing: %s", err)
	}
	for {
		var response hyper_proto.CreateVmResponse
		if err := conn.Decode(&response); err != nil {
			return zeroResponse, fmt.Errorf("error decoding: %s", err)
		}
		if response.Error != "" {
			return zeroResponse, errors.New(response.Error)
		}
		if response.ProgressMessage != "" {
			logger.Debugln(0, response.ProgressMessage)
		}
		if response.Final {
			return response, nil
		}
	}
}

func (r *wrappedReadCloser) Close() error {
	return r.real.Close()
}

func (r *wrappedReadCloser) Read(p []byte) (n int, err error) {
	return r.wrap.Read(p)
}
