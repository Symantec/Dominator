package util

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
)

const procNetRoute = "/proc/net/route"

func getDefaultRoute() (*DefaultRouteInfo, error) {
	if routeTable, err := GetRouteTable(); err != nil {
		return nil, err
	} else if routeTable.DefaultRoute == nil {
		return nil, errors.New("could not find default route")
	} else {
		return routeTable.DefaultRoute, nil
	}
}

func getRouteTable() (*RouteTable, error) {
	file, err := os.Open(procNetRoute)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	var allRoutes, routes []*RouteEntry
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
		routeEntry := &RouteEntry{
			BaseAddr:      intToIP(destAddr),
			BroadcastAddr: intToIP(destAddr | (0xffffffff ^ mask)),
			Flags:         flags,
			InterfaceName: interfaceName,
			Mask:          net.IPMask(intToIP(mask)),
		}
		if flags&RouteFlagGateway != 0 {
			routeEntry.GatewayAddr = intToIP(gatewayAddr)
		}
		allRoutes = append(allRoutes, routeEntry)
		if destAddr == 0 &&
			flags&RouteFlagGateway != 0 &&
			flags&RouteFlagUp != 0 {
			defaultRouteAddr = intToIP(gatewayAddr)
			defaultRouteInterface = interfaceName
			continue
		}
		if destAddr != 0 && flags == RouteFlagUp && gatewayAddr == 0 {
			routes = append(routes, routeEntry)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(defaultRouteAddr) == 0 {
		return &RouteTable{RouteEntries: allRoutes}, nil
	}
	defaultRoute := &DefaultRouteInfo{
		Address:   defaultRouteAddr,
		Interface: defaultRouteInterface,
	}
	for _, route := range routes {
		if route.InterfaceName == defaultRouteInterface &&
			defaultRouteAddr.Mask(route.Mask).Equal(route.BaseAddr) {
			defaultRoute.Mask = route.Mask
			break
		}
	}
	return &RouteTable{DefaultRoute: defaultRoute, RouteEntries: allRoutes}, nil
}

func intToIP(ip uint32) net.IP {
	result := make(net.IP, 4)
	for i := range result {
		result[i] = byte(ip >> uint(8*i))
	}
	return result
}
