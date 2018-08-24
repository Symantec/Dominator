package main

import (
	"bufio"
	"encoding/gob"
	"fmt"
	"io"
	"net"
	"os"

	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/log"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
)

func replaceVmUserDataSubcommand(args []string, logger log.DebugLogger) {
	if err := replaceVmUserData(args[0], logger); err != nil {
		fmt.Fprintf(os.Stderr, "Error replacing VM user data: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func replaceVmUserData(vmHostname string, logger log.DebugLogger) error {
	if vmIP, hypervisor, err := lookupVmAndHypervisor(vmHostname); err != nil {
		return err
	} else {
		return replaceVmUserDataOnHypervisor(hypervisor, vmIP, logger)
	}
}

func replaceVmUserDataOnHypervisor(hypervisor string, ipAddr net.IP,
	logger log.DebugLogger) error {
	if *userDataFile == "" {
		return errors.New("no user data file specified")
	}
	file, size, err := getReader(*userDataFile)
	if err != nil {
		return err
	}
	defer file.Close()
	request := proto.ReplaceVmUserDataRequest{
		IpAddress: ipAddr,
		Size:      uint64(size),
	}
	userDataReader := bufio.NewReader(io.LimitReader(file, size))
	client, err := dialHypervisor(hypervisor)
	if err != nil {
		return err
	}
	defer client.Close()
	conn, err := client.Call("Hypervisor.ReplaceVmUserData")
	if err != nil {
		return err
	}
	defer conn.Close()
	encoder := gob.NewEncoder(conn)
	decoder := gob.NewDecoder(conn)
	if err := encoder.Encode(request); err != nil {
		return err
	}
	logger.Debugln(0, "uploading user data")
	if _, err := io.Copy(conn, userDataReader); err != nil {
		return err
	}
	if err := conn.Flush(); err != nil {
		return err
	}
	var response proto.ReplaceVmUserDataResponse
	if err := decoder.Decode(&response); err != nil {
		return err
	}
	return errors.New(response.Error)
}
