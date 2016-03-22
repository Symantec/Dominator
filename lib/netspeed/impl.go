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
	speedPathFormat          = "/sys/class/net/%s/speed"
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
	file, err := os.Open(fmt.Sprintf(speedPathFormat, interfaceName))
	if err != nil {
		return 0, false
	}
	defer file.Close()
	var value uint64
	nScanned, err := fmt.Fscanf(file, "%d", &value)
	if err != nil || nScanned < 1 {
		return 0, false
	}
	speed = value * 1000000 / 8
	lock.Lock()
	hostToSpeed[hostname] = speed
	lock.Unlock()
	return speed, true
}

func findInterfaceForHost(hostname string) (string, error) {
	var hostIP net.IP
	if hostname == "" {
		hostIP = make(net.IP, 4)
	} else {
		hostIPs, err := net.LookupIP(hostname)
		if err != err {
			return "", err
		}
		if len(hostIPs) < 1 {
			return "", errors.New("not enough IPs")
		}
		hostIP = hostIPs[0]
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
		if hostIP.Mask(maskIP).Equal(destIP) {
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
