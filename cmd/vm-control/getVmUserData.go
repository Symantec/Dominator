package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"

	"github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	proto "github.com/Cloud-Foundations/Dominator/proto/hypervisor"
)

func getVmUserDataSubcommand(args []string, logger log.DebugLogger) error {
	if err := getVmUserData(args[0], logger); err != nil {
		return fmt.Errorf("Error getting VM user data: %s", err)
	}
	return nil
}

func getVmUserData(vmHostname string, logger log.DebugLogger) error {
	if vmIP, hypervisor, err := lookupVmAndHypervisor(vmHostname); err != nil {
		return err
	} else {
		return getVmUserDataOnHypervisor(hypervisor, vmIP, logger)
	}
}

func getVmUserDataOnHypervisor(hypervisor string, ipAddr net.IP,
	logger log.DebugLogger) error {
	if *userDataFile == "" {
		return errors.New("no user data file specified")
	}
	client, err := dialHypervisor(hypervisor)
	if err != nil {
		return err
	}
	defer client.Close()
	conn, err := client.Call("Hypervisor.GetVmUserData")
	if err != nil {
		return err
	}
	defer conn.Close()
	request := proto.GetVmUserDataRequest{IpAddress: ipAddr}
	if err := conn.Encode(request); err != nil {
		return err
	}
	if err := conn.Flush(); err != nil {
		return err
	}
	var response proto.GetVmUserDataResponse
	if err := conn.Decode(&response); err != nil {
		return err
	}
	if err := errors.New(response.Error); err != nil {
		return err
	}
	if response.Length < 1 {
		return errors.New("no user data")
	}
	file, err := os.OpenFile(*userDataFile, os.O_WRONLY|os.O_CREATE,
		privateFilePerms)
	if err != nil {
		io.CopyN(ioutil.Discard, conn, int64(response.Length))
		return err
	}
	defer file.Close()
	logger.Debugln(0, "downloading user data")
	if _, err := io.CopyN(file, conn, int64(response.Length)); err != nil {
		return err
	}
	return errors.New(response.Error)
}
