package util

import (
	"bufio"
	"errors"
	"net"
	"os"
	"strings"
)

const etcResolvConf = "/etc/resolv.conf"

func getResolverConfiguration() (*ResolverConfiguration, error) {
	file, err := os.Open(etcResolvConf)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	resolverConfiguration := &ResolverConfiguration{}
	for scanner.Scan() {
		splitLine := strings.Split(scanner.Text(), ";")
		if len(splitLine) < 1 {
			continue
		}
		if splitLine[0] == "" {
			continue
		}
		fields := strings.Fields(splitLine[0])
		switch fields[0] {
		case "domain":
			resolverConfiguration.Domain = fields[1]
		case "nameserver":
			if len(fields) < 2 {
				return nil, errors.New("missing nameserver: " + splitLine[0])
			}
			for _, nameserver := range fields[1:] {
				if addr := net.ParseIP(nameserver); addr == nil {
					return nil, errors.New("bad address: " + splitLine[0])
				} else {
					resolverConfiguration.Nameservers = append(
						resolverConfiguration.Nameservers, shrinkIP(addr))
				}
			}
		case "search":
			resolverConfiguration.SearchDomains = append(
				resolverConfiguration.SearchDomains, fields[1:]...)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return resolverConfiguration, nil
}

func shrinkIP(netIP net.IP) net.IP {
	switch len(netIP) {
	case 4:
		return netIP
	case 16:
		if ip4 := netIP.To4(); ip4 == nil {
			return netIP
		} else {
			return ip4
		}
	default:
		return netIP
	}
}
