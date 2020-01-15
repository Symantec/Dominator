package main

import (
	"fmt"

	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/dominator"
)

func configureSubsSubcommand(args []string, logger log.DebugLogger) error {
	if err := configureSubs(getClient()); err != nil {
		return fmt.Errorf("Error setting config for subs: %s", err)
	}
	return nil
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
