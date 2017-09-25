package main

import (
	"os"

	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/sub/client"
)

func boostCpuLimitSubcommand(getSubClient getSubClientFunc, args []string) {
	if err := boostCpuLimit(getSubClient()); err != nil {
		logger.Fatalf("Error boosting CPU limit: %s\n", err)
	}
	os.Exit(0)
}

func boostCpuLimit(srpcClient *srpc.Client) error {
	return client.BoostCpuLimit(srpcClient)
}
