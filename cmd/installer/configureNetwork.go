// +build linux

package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/Cloud-Foundations/Dominator/lib/fsutil"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	libnet "github.com/Cloud-Foundations/Dominator/lib/net"
	"github.com/Cloud-Foundations/Dominator/lib/net/configurator"
	fm_proto "github.com/Cloud-Foundations/Dominator/proto/fleetmanager"
)

func addMapping(mappings map[string]string, name string) error {
	filename := fmt.Sprintf("/sys/class/net/%s/device", name)
	if symlink, err := os.Readlink(filename); err != nil {
		return err
	} else {
		mappings[name] = filepath.Base(symlink)
		return nil
	}
}

func configureNetwork(machineInfo fm_proto.GetMachineInfoResponse,
	interfaces map[string]net.Interface, logger log.DebugLogger) error {
	hostname := strings.Split(machineInfo.Machine.Hostname, ".")[0]
	err := ioutil.WriteFile(filepath.Join(*mountPoint, "etc", "hostname"),
		[]byte(hostname+"\n"), fsutil.PublicFilePerms)
	if err != nil {
		return err
	}
	netconf, err := configurator.Compute(machineInfo,
		markConnectedInterfaces(interfaces, logger), logger)
	if err != nil {
		return err
	}
	mappings := make(map[string]string)
	for name := range interfaces {
		if err := addMapping(mappings, name); err != nil {
			return err
		}
	}
	if !*dryRun {
		if err := netconf.WriteDebian(*mountPoint); err != nil {
			return err
		}
		if err := writeMappings(mappings); err != nil {
			return err
		}
		err = configurator.WriteResolvConf(*mountPoint, netconf.DefaultSubnet)
		if err != nil {
			return err
		}
	}
	return nil
}

// Return a new map of interfaces, marking those with a carrier as up and those
// without a carrier as down.
func markConnectedInterfaces(interfaces map[string]net.Interface,
	logger log.DebugLogger) map[string]net.Interface {
	outputInterfaces := make(map[string]net.Interface)
	for name, iface := range interfaces {
		if libnet.TestCarrier(name) {
			iface.Flags |= net.FlagUp
			logger.Debugf(1, "%s is connected\n", name)
		} else {
			iface.Flags &= ^net.FlagUp
			run("ifconfig", "", logger, name, "down")
		}
		outputInterfaces[name] = iface
	}
	return outputInterfaces
}

func writeMappings(mappings map[string]string) error {
	filename := filepath.Join(*mountPoint,
		"etc", "udev", "rules.d", "70-persistent-net.rules")
	if file, err := create(filename); err != nil {
		return err
	} else {
		defer file.Close()
		writer := bufio.NewWriter(file)
		defer writer.Flush()
		for name, kernelId := range mappings {
			fmt.Fprintf(writer,
				`SUBSYSTEM=="net", ACTION=="add", DRIVERS=="?*", ATTR{type}=="1", KERNELS=="%s", NAME="%s"`,
				kernelId, name)
			fmt.Fprintln(writer)
		}
		return writer.Flush()
	}
}
