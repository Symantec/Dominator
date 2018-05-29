package main

import (
	"fmt"
	"net"
	"os"

	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
)

func snapshotVmSubcommand(args []string, logger log.DebugLogger) {
	if err := snapshotVm(args[0], logger); err != nil {
		fmt.Fprintf(os.Stderr, "Error snapshotting VM: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func snapshotVm(ipAddr string, logger log.DebugLogger) error {
	vmIP := net.ParseIP(ipAddr)
	if hypervisor, err := findHypervisor(vmIP); err != nil {
		return err
	} else {
		return snapshotVmOnHypervisor(hypervisor, vmIP, logger)
	}
}

func snapshotVmOnHypervisor(hypervisor string, ipAddr net.IP,
	logger log.DebugLogger) error {
	request := proto.SnapshotVmRequest{ipAddr, *forceIfNotStopped,
		*snapshotRootOnly}
	client, err := srpc.DialHTTP("tcp", hypervisor, 0)
	if err != nil {
		return err
	}
	defer client.Close()
	var reply proto.SnapshotVmResponse
	err = client.RequestReply("Hypervisor.SnapshotVm", request, &reply)
	if err != nil {
		return err
	}
	return errors.New(reply.Error)
}
