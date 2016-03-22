package netspeed

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
)

const (
	procRouteFilename string = "/proc/net/route"
)

var (
	lock        sync.Mutex
	hostToSpeed = make(map[string]uint64)
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
	lock.Lock()
	speed, ok := hostToSpeed[hostname]
	lock.Unlock()
	if ok {
		return speed, true
	}
	interfaceName, err := findInterfaceForHost(hostname)
	if err != nil {
		return 0, false
	}
	_ = interfaceName
	return 0, false
}

func findInterfaceForHost(hostname string) (string, error) {
	hostIPs, err := net.LookupIP(hostname)
	if err != err {
		return "", err
	}
	if len(hostIPs) < 1 {
		return "", errors.New("not enough IPs")
	}
	file, err := os.Open(procRouteFilename)
	if err != nil {
		return "", err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	mostSpecificInterfaceName := ""
	mostSpecificNetmaskBits := -1
	for scanner.Scan() {
		var interfaceName string
		var destAddr, gatewayAddr, flags, mask uint32
		var ign int
		nCopied, err := fmt.Sscanf(scanner.Text(),
			"%s %x %x %x %d %d %d %x %d %d %d",
			&interfaceName, &destAddr, &gatewayAddr, &flags, &ign, &ign, &ign,
			&mask, &ign, &ign, &ign)
		if err != nil || nCopied < 11 {
			continue
		}
		maskIP := net.IPMask(intToIP(mask))
		destIP := intToIP(destAddr)
		if hostIPs[0].Mask(maskIP).Equal(destIP) {
			size, _ := maskIP.Size()
			if size > mostSpecificNetmaskBits {
				mostSpecificInterfaceName = interfaceName
				mostSpecificNetmaskBits = size
			}
		}
	}
	return mostSpecificInterfaceName, scanner.Err()
}

func intToIP(ip uint32) net.IP {
	result := make(net.IP, 4)
	for i, _ := range result {
		result[i] = byte(ip >> uint(8*i))
	}
	return result
}
