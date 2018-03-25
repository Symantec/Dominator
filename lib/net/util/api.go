package util

import (
	"net"
)

type DefaultRouteInfo struct {
	Address   net.IP
	Interface string
	Mask      net.IPMask
}

type ResolverConfiguration struct {
	Domain        string
	Nameservers   []net.IP
	SearchDomains []string
}

func GetDefaultRoute() (*DefaultRouteInfo, error) {
	return getDefaultRoute()
}

func GetMyIP() (net.IP, error) {
	return getMyIP()
}

func GetResolverConfiguration() (*ResolverConfiguration, error) {
	return getResolverConfiguration()
}
