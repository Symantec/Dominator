package util

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
)

const procNetRoute = "/proc/net/route"

type routeInfo struct {
	interfaceName string
	destAddr      net.IP
	mask          net.IPMask
}

func getDefaultRoute() (*DefaultRouteInfo, error) {
	file, err := os.Open(procNetRoute)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	routes := make([]routeInfo, 0)
	var defaultRouteAddr net.IP
	var defaultRouteInterface string
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
		if destAddr == 0 && flags&0x2 == 0x2 && flags&0x1 == 0x1 {
			defaultRouteAddr = intToIP(gatewayAddr)
			defaultRouteInterface = interfaceName
			continue
		}
		if destAddr != 0 && flags == 0x1 && gatewayAddr == 0 {
			routes = append(routes, routeInfo{
				interfaceName: interfaceName,
				destAddr:      intToIP(destAddr),
				mask:          net.IPMask(intToIP(mask)),
			})
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(defaultRouteAddr) == 0 {
		return nil, errors.New("could not find default route")
	}
	defaultRoute := &DefaultRouteInfo{
		Address:   defaultRouteAddr,
		Interface: defaultRouteInterface,
	}
	for _, route := range routes {
		if route.interfaceName == defaultRouteInterface &&
			defaultRouteAddr.Mask(route.mask).Equal(route.destAddr) {
			defaultRoute.Mask = route.mask
			break
		}
	}
	return defaultRoute, nil
}

func intToIP(ip uint32) net.IP {
	result := make(net.IP, 4)
	for i := range result {
		result[i] = byte(ip >> uint(8*i))
	}
	return result
}
