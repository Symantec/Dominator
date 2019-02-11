package main

import (
	"encoding/gob"
	"errors"
	"fmt"
	"net"
	"os"

	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
	hyper_proto "github.com/Symantec/Dominator/proto/hypervisor"
)

func copyVmSubcommand(args []string, logger log.DebugLogger) {
	if err := copyVm(args[0], logger); err != nil {
		fmt.Fprintf(os.Stderr, "Error copying VM: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func copyVm(vmHostname string, logger log.DebugLogger) error {
	if vmIP, hypervisor, err := searchVmAndHypervisor(vmHostname); err != nil {
		return err
	} else {
		return copyVmFromHypervisor(hypervisor, vmIP, logger)
	}
}

func callCopyVm(client *srpc.Client, request hyper_proto.CopyVmRequest,
	reply *hyper_proto.CopyVmResponse, logger log.DebugLogger) error {
	conn, err := client.Call("Hypervisor.CopyVm")
	if err != nil {
		return err
	}
	defer conn.Close()
	encoder := gob.NewEncoder(conn)
	decoder := gob.NewDecoder(conn)
	if err := encoder.Encode(request); err != nil {
		return err
	}
	if err := conn.Flush(); err != nil {
		return err
	}
	for {
		var response hyper_proto.CopyVmResponse
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

func copyVmFromHypervisor(sourceHypervisorAddress string, vmIP net.IP,
	logger log.DebugLogger) error {
	destHypervisorAddress, err := getHypervisorAddress()
	if err != nil {
		return err
	}
	sourceHypervisor, err := dialHypervisor(sourceHypervisorAddress)
	if err != nil {
		return err
	}
	defer sourceHypervisor.Close()
	accessToken, err := getVmAccessTokenClient(sourceHypervisor, vmIP)
	if err != nil {
		return err
	}
	defer discardAccessToken(sourceHypervisor, vmIP)
	destHypervisor, err := dialHypervisor(destHypervisorAddress)
	if err != nil {
		return err
	}
	defer destHypervisor.Close()
	request := hyper_proto.CopyVmRequest{
		AccessToken:      accessToken,
		IpAddress:        vmIP,
		SourceHypervisor: sourceHypervisorAddress,
	}
	var reply hyper_proto.CopyVmResponse
	logger.Debugf(0, "copying VM to %s\n", destHypervisorAddress)
	if err := callCopyVm(destHypervisor, request, &reply, logger); err != nil {
		return err
	}
	if err := acknowledgeVm(destHypervisor, reply.IpAddress); err != nil {
		return fmt.Errorf("error acknowledging VM: %s", err)
	}
	fmt.Println(reply.IpAddress)
	return nil
}
