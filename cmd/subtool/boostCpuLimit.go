package main

import (
	"fmt"

	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/sub/client"
)

func boostCpuLimitSubcommand(args []string, logger log.DebugLogger) error {
	srpcClient := getSubClient(logger)
	defer srpcClient.Close()
	if err := boostCpuLimit(srpcClient); err != nil {
		return fmt.Errorf("Error boosting CPU limit: %s", err)
	}
	return nil
}

func boostCpuLimit(srpcClient *srpc.Client) error {
	return client.BoostCpuLimit(srpcClient)
}
