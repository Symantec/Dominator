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
	fmCR := srpc.NewClientResource("tcp",
		fmt.Sprintf("%s:%d", *fleetManagerHostname, *fleetManagerPortNum))
	defer fmCR.ScheduleClose()
	if info, err := getInfoForMachine(fmCR, hostname); err != nil {
		return err
	} else {
		return json.WriteWithIndent(os.Stdout, "    ", info)
	}
}

func getInfoForMachine(fmCR *srpc.ClientResource, hostname string) (
	proto.GetMachineInfoResponse, error) {
	request := proto.GetMachineInfoRequest{Hostname: hostname}
	var reply proto.GetMachineInfoResponse
	client, err := fmCR.GetHTTP(nil, 0)
	if err != nil {
		return proto.GetMachineInfoResponse{}, err
	}
	defer client.Put()
	err = client.RequestReply("FleetManager.GetMachineInfo", request, &reply)
	if err != nil {
		return proto.GetMachineInfoResponse{}, err
	}
	if err := errors.New(reply.Error); err != nil {
		return proto.GetMachineInfoResponse{}, err
	}
	return reply, nil
}
