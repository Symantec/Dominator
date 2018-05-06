package main

import (
	"fmt"
	"net"

	"github.com/Symantec/Dominator/lib/errors"
)

func findHypervisor(vmIpAddr net.IP) (string, error) {
	if *hypervisorHostname != "" {
		return fmt.Sprintf("%s:%d", *hypervisorHostname, *hypervisorPortNum),
			nil
	} else {
		return "", errors.New("no Hypervisor specified")
	}
}
