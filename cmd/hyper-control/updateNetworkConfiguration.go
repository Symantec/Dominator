package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/Symantec/Dominator/lib/log"
)

func updateNetworkConfigurationSubcommand(args []string,
	logger log.DebugLogger) {
	err := updateNetworkConfiguration(logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error updating network configuration: %s\n",
			err)
		os.Exit(1)
	}
	os.Exit(0)
}

func updateNetworkConfiguration(logger log.DebugLogger) error {
	netconf, err := getNetworkConfiguration(logger)
	if err != nil {
		return err
	}
	if changed, err := netconf.Update("/", logger); err != nil {
		return err
	} else if !changed {
		return nil
	}
	logger.Println("restarting hypervisor")
	cmd := exec.Command("service", "hypervisor", "restart")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
