package main

import (
	"fmt"
	"os"

	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/dominator"
)

func configureSubsSubcommand(client *srpc.Client, args []string) {
	if err := configureSubs(client); err != nil {
		fmt.Fprintf(os.Stderr, "Error setting config for subs: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func configureSubs(client *srpc.Client) error {
	var request dominator.ConfigureSubsRequest
	var reply dominator.ConfigureSubsResponse
	request.CpuPercent = *cpuPercent
	request.NetworkSpeedPercent = *networkSpeedPercent
	request.ScanExclusionList = scanExcludeList
	request.ScanSpeedPercent = *scanSpeedPercent
	return client.RequestReply("Dominator.ConfigureSubs", request, &reply)
}
