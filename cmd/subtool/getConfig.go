package main

import (
	"fmt"
	"github.com/Symantec/Dominator/proto/sub"
	"net/rpc"
	"os"
)

func getConfigSubcommand(client *rpc.Client, args []string) {
	err := getConfig(client)
	if err != nil {
		fmt.Printf("Error getting config\t%s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func getConfig(client *rpc.Client) error {
	var request sub.GetConfigurationRequest
	var reply sub.GetConfigurationResponse
	err := client.Call("Subd.GetConfiguration", request, &reply)
	if err != nil {
		return err
	}
	fmt.Println(reply)
	return nil
}
