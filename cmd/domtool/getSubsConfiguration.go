package main

import (
	"fmt"

	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/dominator"
	"github.com/Cloud-Foundations/Dominator/proto/sub"
)

func getSubsConfigurationSubcommand(args []string,
	logger log.DebugLogger) error {
	if err := getSubsConfiguration(getClient()); err != nil {
		return fmt.Errorf("Error getting config for subs: %s", err)
	}
	return nil
}

func getSubsConfiguration(client *srpc.Client) error {
	var request dominator.GetSubsConfigurationRequest
	var reply dominator.GetSubsConfigurationResponse
	if err := client.RequestReply("Dominator.GetSubsConfiguration", request,
		&reply); err != nil {
		return err
	}
	fmt.Println(sub.Configuration(reply))
	return nil
}
