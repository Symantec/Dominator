package main

import (
	"encoding/gob"
	"fmt"
	"os"

	"github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
)

func getUpdatesSubcommand(args []string, logger log.DebugLogger) {
	if err := getUpdates(logger); err != nil {
		fmt.Fprintf(os.Stderr, "Error getting updates: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func getUpdates(logger log.DebugLogger) error {
	hypervisor := fmt.Sprintf("%s:%d", *hypervisorHostname, *hypervisorPortNum)
	return getUpdatesOnHypervisor(hypervisor, logger)
}

func getUpdatesOnHypervisor(hypervisor string, logger log.DebugLogger) error {
	client, err := srpc.DialHTTP("tcp", hypervisor, 0)
	if err != nil {
		return err
	}
	defer client.Close()
	conn, err := client.Call("Hypervisor.GetUpdates")
	if err != nil {
		return err
	}
	defer conn.Close()
	decoder := gob.NewDecoder(conn)
	for {
		var update proto.Update
		if err := decoder.Decode(&update); err != nil {
			return err
		}
		if err := json.WriteWithIndent(os.Stdout, "    ", update); err != nil {
			return err
		}
	}
	return nil
}
