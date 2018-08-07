package main

import (
	"fmt"
	"os"

	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/fleetmanager"
)

func getMachineInfoSubcommand(args []string, logger log.DebugLogger) {
	err := getMachineInfo(args[0], logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting machine info: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func getMachineInfo(hostname string, logger log.DebugLogger) error {
	request := proto.GetMachineInfoRequest{Hostname: hostname}
	var reply proto.GetMachineInfoResponse
	clientName := fmt.Sprintf("%s:%d",
		*fleetManagerHostname, *fleetManagerPortNum)
	client, err := srpc.DialHTTP("tcp", clientName, 0)
	if err != nil {
		return err
	}
	defer client.Close()
	err = client.RequestReply("FleetManager.GetMachineInfo", request, &reply)
	if err != nil {
		return err
	}
	if err := errors.New(reply.Error); err != nil {
		return err
	}
	return json.WriteWithIndent(os.Stdout, "    ", reply)
}
