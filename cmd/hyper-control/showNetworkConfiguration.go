package main

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/Cloud-Foundations/Dominator/lib/json"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	libnet "github.com/Cloud-Foundations/Dominator/lib/net"
	"github.com/Cloud-Foundations/Dominator/lib/net/configurator"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	fm_proto "github.com/Cloud-Foundations/Dominator/proto/fleetmanager"
)

type networkInterface struct {
	HardwareAddr string
	Name         string
	Up           bool
}

func showNetworkConfigurationSubcommand(args []string,
	logger log.DebugLogger) error {
	err := showNetworkConfiguration(logger)
	if err != nil {
		return fmt.Errorf("Error showing network configuration: %s", err)
	}
	return nil
}

func getInfoForhost(hostname string) (fm_proto.GetMachineInfoResponse, error) {
	fmCR := srpc.NewClientResource("tcp",
		fmt.Sprintf("%s:%d", *fleetManagerHostname, *fleetManagerPortNum))
	defer fmCR.ScheduleClose()
	if hostname != "" && hostname != "localhost" {
		return getInfoForMachine(fmCR, hostname)
	}
	if hostname, err := os.Hostname(); err != nil {
		return fm_proto.GetMachineInfoResponse{}, err
	} else if info, err := getInfoForMachine(fmCR, hostname); err == nil {
		return info, nil
	} else if !strings.Contains(err.Error(), "unknown machine") {
		return fm_proto.GetMachineInfoResponse{}, err
	} else if hostname, err := getHostname(); err != nil {
		return fm_proto.GetMachineInfoResponse{}, err
	} else {
		return getInfoForMachine(fmCR, hostname)
	}
}

func getNetworkConfiguration(hostname string, logger log.DebugLogger) (
	*configurator.NetworkConfig, error) {
	info, err := getInfoForhost(hostname)
	if err != nil {
		return nil, err
	}
	var interfacesMap map[string]net.Interface
	if *networkInterfacesFile == "" {
		if hostname != "" && hostname != "localhost" {
			return nil, errors.New("no networkInterfacesFile specified")
		}
		_, interfacesMap, err = getUpInterfaces(logger)
		if err != nil {
			return nil, err
		}
	} else {
		var networkInterfaces []networkInterface
		err := json.ReadFromFile(*networkInterfacesFile, &networkInterfaces)
		if err != nil {
			return nil, err
		}
		interfacesMap = make(map[string]net.Interface, len(networkInterfaces))
		for _, netInterface := range networkInterfaces {
			macAddress, err := net.ParseMAC(netInterface.HardwareAddr)
			if err != nil {
				return nil, err
			}
			netIf := net.Interface{
				Name:         netInterface.Name,
				HardwareAddr: macAddress,
			}
			if netInterface.Up {
				netIf.Flags = net.FlagUp
			}
			interfacesMap[netInterface.Name] = netIf
		}
	}
	return configurator.Compute(info, interfacesMap, logger)
}

func getUpInterfaces(logger log.DebugLogger) (
	[]net.Interface, map[string]net.Interface, error) {
	interfaceList, interfaceMap, err := libnet.ListBroadcastInterfaces(
		libnet.InterfaceTypeEtherNet, logger)
	if err != nil {
		return nil, nil, err
	}
	newList := make([]net.Interface, 0, len(interfaceList))
	for _, iface := range interfaceList {
		if libnet.TestCarrier(iface.Name) {
			newList = append(newList, iface)
			logger.Debugf(1, "will generate configuration including: %s\n",
				iface.Name)
		} else {
			delete(interfaceMap, iface.Name)
		}
	}
	return newList, interfaceMap, nil
}

func showNetworkConfiguration(logger log.DebugLogger) error {
	netconf, err := getNetworkConfiguration(*hypervisorHostname, logger)
	if err != nil {
		return err
	}
	if netconf.DefaultSubnet == nil {
		return errors.New("no default subnet found")
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
