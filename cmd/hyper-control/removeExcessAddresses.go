package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
)

func removeExcessAddressesSubcommand(args []string, logger log.DebugLogger) {
	err := removeExcessAddresses(args[0], logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error removing excess addresses: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func removeExcessAddresses(maxAddr string, logger log.DebugLogger) error {
	maxAddresses, err := strconv.ParseUint(maxAddr, 10, 16)
	if err != nil {
		return err
	}
	request := proto.ChangeAddressPoolRequest{
		MaximumFreeAddresses: map[string]uint{"": uint(maxAddresses)}}
	var reply proto.ChangeAddressPoolResponse
	clientName := fmt.Sprintf("%s:%d", *hypervisorHostname, *hypervisorPortNum)
	client, err := srpc.DialHTTP("tcp", clientName, 0)
	if err != nil {
		return err
	}
	defer client.Close()
	err = client.RequestReply("Hypervisor.ChangeAddressPool",
		request, &reply)
	if err != nil {
		return err
	}
	return errors.New(reply.Error)
}
