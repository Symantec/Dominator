package main

import (
	"bufio"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"net"
	"os"

	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
)

func replaceVmImageSubcommand(args []string, logger log.DebugLogger) {
	if err := replaceVmImage(args[0], logger); err != nil {
		fmt.Fprintf(os.Stderr, "Error replacing VM image: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func callReplaceVmImage(client *srpc.Client,
	request proto.ReplaceVmImageRequest, reply *proto.ReplaceVmImageResponse,
	imageReader io.Reader, logger log.DebugLogger) error {
	conn, err := client.Call("Hypervisor.ReplaceVmImage")
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
	if err := conn.Flush(); err != nil {
		return err
	}
	for {
		var response proto.ReplaceVmImageResponse
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

func replaceVmImage(ipAddr string, logger log.DebugLogger) error {
	vmIP := net.ParseIP(ipAddr)
	if hypervisor, err := findHypervisor(vmIP); err != nil {
		return err
	} else {
		return replaceVmImageOnHypervisor(hypervisor, vmIP, logger)
	}
}

func replaceVmImageOnHypervisor(hypervisor string, ipAddr net.IP,
	logger log.DebugLogger) error {
	request := proto.ReplaceVmImageRequest{
		DhcpTimeout:      *responseTimeout,
		IpAddress:        ipAddr,
		MinimumFreeBytes: *minFreeBytes,
		RoundupPower:     *roundupPower,
	}
	var imageReader io.Reader
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
	client, err := srpc.DialHTTP("tcp", hypervisor, 0)
	if err != nil {
		return err
	}
	defer client.Close()
	var reply proto.ReplaceVmImageResponse
	err = callReplaceVmImage(client, request, &reply, imageReader, logger)
	if err != nil {
		return err
	}
	if reply.DhcpTimedOut {
		return errors.New("DHCP ACK timed out")
	}
	return nil
}
