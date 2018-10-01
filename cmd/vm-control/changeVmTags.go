package main

import (
	"fmt"
	"net"
	"os"

	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/lib/tags"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
)

func changeVmTagsSubcommand(args []string, logger log.DebugLogger) {
	if err := changeVmTags(args[0], logger); err != nil {
		fmt.Fprintf(os.Stderr, "Error changing VM tags: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func changeVmTags(vmHostname string, logger log.DebugLogger) error {
	if vmIP, hypervisor, err := lookupVmAndHypervisor(vmHostname); err != nil {
		return err
	} else {
		return changeVmTagsOnHypervisor(hypervisor, vmIP, logger)
	}
}

func changeVmTagsOnHypervisor(hypervisor string, ipAddr net.IP,
	logger log.DebugLogger) error {
	client, err := dialHypervisor(hypervisor)
	if err != nil {
		return err
	}
	defer client.Close()
	if _, ok := vmTags[""]; ok {
		return setVmTagsOnHypervisor(client, ipAddr, vmTags, logger)
	}
	if _, ok := vmTags["*"]; ok {
		return setVmTagsOnHypervisor(client, ipAddr, vmTags, logger)
	}
	request := proto.GetVmInfoRequest{ipAddr}
	var reply proto.GetVmInfoResponse
	err = client.RequestReply("Hypervisor.GetVmInfo", request, &reply)
	if err != nil {
		return err
	}
	if err := errors.New(reply.Error); err != nil {
		return err
	}
	reply.VmInfo.Tags.Merge(vmTags)
	for key, value := range reply.VmInfo.Tags {
		if value == "" {
			delete(reply.VmInfo.Tags, key)
		}
	}
	return setVmTagsOnHypervisor(client, ipAddr, reply.VmInfo.Tags, logger)
}

func setVmTagsOnHypervisor(client *srpc.Client, ipAddr net.IP,
	vmTags tags.Tags, logger log.DebugLogger) error {
	request := proto.ChangeVmTagsRequest{ipAddr, vmTags}
	var reply proto.ChangeVmTagsResponse
	err := client.RequestReply("Hypervisor.ChangeVmTags", request, &reply)
	if err != nil {
		return err
	}
	return errors.New(reply.Error)
}
