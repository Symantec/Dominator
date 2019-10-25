package main

import (
	"fmt"
	"os"

	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/dominator"
	"github.com/Cloud-Foundations/Dominator/proto/sub"
)

func getSubsConfigurationSubcommand(client *srpc.Client, args []string) {
	if err := getSubsConfiguration(client); err != nil {
		fmt.Fprintf(os.Stderr, "Error getting config for subs: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
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
