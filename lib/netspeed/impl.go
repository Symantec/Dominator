package netspeed

import (
	"strings"
)

func getSpeedToAddress(address string) (uint64, bool) {
	if fields := strings.Split(address, ":"); len(fields) == 2 {
		return getSpeedToHost(fields[0])
	}
	return 0, false
}

func getSpeedToHost(hostname string) (uint64, bool) {
	if hostname == "localhost" {
		return 0, true
	}
	return 0, false
}
