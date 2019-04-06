package main

import (
	"fmt"
	"os"

	"github.com/Symantec/Dominator/lib/log"
	libnet "github.com/Symantec/Dominator/lib/net"
	"github.com/Symantec/Dominator/lib/net/configurator"
	"github.com/Symantec/Dominator/lib/srpc"
)

func showNetworkConfigurationSubcommand(args []string,
	logger log.DebugLogger) error {
	err := showNetworkConfiguration(logger)
	if err != nil {
		return fmt.Errorf("Error showing network configuration: %s", err)
	}
	return nil
}

func getNetworkConfiguration(logger log.DebugLogger) (
	*configurator.NetworkConfig, error) {
	fmCR := srpc.NewClientResource("tcp",
		fmt.Sprintf("%s:%d", *fleetManagerHostname, *fleetManagerPortNum))
	defer fmCR.ScheduleClose()
	_, interfaces, err := libnet.ListBroadcastInterfaces(
		libnet.InterfaceTypeEtherNet, logger)
	if err != nil {
		return nil, err
	}
	for name := range interfaces {
		if libnet.TestCarrier(name) {
			logger.Debugf(1, "will generate configuration including: %s\n",
				name)
		} else {
			delete(interfaces, name)
		}
	}
	hostname, err := getHostname()
	if err != nil {
		return nil, err
	}
	info, err := getInfoForMachine(fmCR, hostname)
	if err != nil {
		return nil, err
	}
	return configurator.Compute(info, interfaces, logger)
}

func showNetworkConfiguration(logger log.DebugLogger) error {
	netconf, err := getNetworkConfiguration(logger)
	if err != nil {
		return err
	}
	fmt.Println("=============================================================")
	fmt.Println("Network configuration:")
	if err := netconf.PrintDebian(os.Stdout); err != nil {
		return err
	}
	fmt.Println("=============================================================")
	fmt.Println("DNS configuration:")
	return configurator.PrintResolvConf(os.Stdout, netconf.DefaultSubnet)
}
