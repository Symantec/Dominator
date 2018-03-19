package main

import (
	"bufio"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"net"
	"os"

	"github.com/Symantec/Dominator/lib/flagutil"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
)

func createVmSubcommand(args []string, logger log.DebugLogger) {
	if err := createVm(logger); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating VM: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func acknowledgeVm(client *srpc.Client, ipAddress net.IP) error {
	request := proto.AcknowledgeVmRequest{ipAddress}
	var reply proto.AcknowledgeVmResponse
	return client.RequestReply("Hypervisor.AcknowledgeVm", request, &reply)
}

func callCreateVm(client *srpc.Client, request proto.CreateVmRequest,
	reply *proto.CreateVmResponse, imageReader, userDataReader io.Reader,
	logger log.DebugLogger) error {
	conn, err := client.Call("Hypervisor.CreateVm")
	if err != nil {
		return err
	}
	defer conn.Close()
	encoder := gob.NewEncoder(conn)
	decoder := gob.NewDecoder(conn)
	if err := encoder.Encode(request); err != nil {
		return err
	}
	// Stream any required data.
	if imageReader != nil {
		logger.Debugln(0, "uploading image")
		if _, err := io.Copy(conn, imageReader); err != nil {
			return err
		}
	}
	if userDataReader != nil {
		logger.Debugln(0, "uploading user data")
		if _, err := io.Copy(conn, userDataReader); err != nil {
			return err
		}
	}
	if err := conn.Flush(); err != nil {
		return err
	}
	for {
		var response proto.CreateVmResponse
		if err := decoder.Decode(&response); err != nil {
			return err
		}
		if response.Error != "" {
			return errors.New(response.Error)
		}
		if response.ProgressMessage != "" {
			logger.Debugln(0, response.ProgressMessage)
		}
		if response.Final {
			*reply = response
			return nil
		}
	}
}

func createVm(logger log.DebugLogger) error {
	hypervisor := fmt.Sprintf("%s:%d", *hypervisorHostname, *hypervisorPortNum)
	return createVmOnHypervisor(hypervisor, logger)
}

func createVmOnHypervisor(hypervisor string, logger log.DebugLogger) error {
	request := proto.CreateVmRequest{
		DhcpTimeout: *responseTimeout,
		VmInfo: proto.VmInfo{
			MemoryInMiB: *memory,
			MilliCPUs:   *milliCPUs,
			OwnerGroups: ownerGroups,
			OwnerUsers:  ownerUsers,
			Tags:        vmTags,
			SubnetId:    *subnetId,
		},
		MinimumFreeBytes: *minFreeBytes,
		RoundupPower:     *roundupPower,
	}
	if sizes, err := parseSizes(secondaryVolumeSizes); err != nil {
		return err
	} else {
		request.SecondaryVolumes = sizes
	}
	var imageReader, userDataReader io.Reader
	if *imageName != "" {
		request.ImageName = *imageName
		request.ImageTimeout = *imageTimeout
	} else if *imageURL != "" {
		request.ImageURL = *imageURL
	} else if *imageFile != "" {
		file, size, err := getReader(*imageFile)
		if err != nil {
			return err
		} else {
			defer file.Close()
			request.ImageDataSize = uint64(size)
			imageReader = bufio.NewReader(io.LimitReader(file, size))
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
			userDataReader = bufio.NewReader(io.LimitReader(file, size))
		}
	}
	client, err := srpc.DialHTTP("tcp", hypervisor, 0)
	if err != nil {
		return err
	}
	defer client.Close()
	var reply proto.CreateVmResponse
	err = callCreateVm(client, request, &reply, imageReader, userDataReader,
		logger)
	if err != nil {
		return err
	}
	if err := acknowledgeVm(client, reply.IpAddress); err != nil {
		return err
	}
	fmt.Println(reply.IpAddress)
	if reply.DhcpTimedOut {
		return errors.New("DHCP ACK timed out")
	}
	return nil
}
func getReader(filename string) (io.ReadCloser, int64, error) {
	if file, err := os.Open(filename); err != nil {
		return nil, -1, err
	} else {
		fi, err := file.Stat()
		if err != nil {
			file.Close()
			return nil, -1, err
		}
		return file, fi.Size(), nil
	}
}

func parseSizes(strSizes flagutil.StringList) ([]proto.Volume, error) {
	var volumes []proto.Volume
	for _, strSize := range strSizes {
		var size uint64
		if _, err := fmt.Sscanf(strSize, "%dM", &size); err == nil {
			volumes = append(volumes, proto.Volume{size << 20})
		} else if _, err := fmt.Sscanf(strSize, "%dG", &size); err == nil {
			volumes = append(volumes, proto.Volume{size << 30})
		} else {
			return nil, err
		}
	}
	return volumes, nil
}
