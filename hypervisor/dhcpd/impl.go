package dhcpd

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/Symantec/Dominator/lib/log"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
	dhcp "github.com/krolaw/dhcp4"
)

const sysClassNet = "/sys/class/net"
const leaseTime = time.Hour * 48

type routeType struct {
	interfaceName string
	destination   uint32
	mask          uint32
}

func newServer(bridges []string, logger log.DebugLogger) (*DhcpServer, error) {
	dhcpServer := &DhcpServer{
		logger:          logger,
		ackChannels:     make(map[string]chan struct{}),
		ipAddrToMacAddr: make(map[string]string),
		leases:          make(map[string]proto.Address),
	}
	if myIP, err := getMyIP(); err != nil {
		return nil, err
	} else {
		dhcpServer.myIP = myIP
	}
	if len(bridges) < 1 {
		logger.Debugf(0, "Starting DHCP server on all interfaces, addr: %s\n",
			dhcpServer.myIP)
		go func() {
			if err := dhcp.ListenAndServe(dhcpServer); err != nil {
				logger.Println(err)
			}
		}()
		return dhcpServer, nil
	}
	for _, bridge := range bridges {
		logger.Debugf(0, "Starting DHCP server on interface: %s, addr: %s\n",
			bridge, dhcpServer.myIP)
		go func(bridge string) {
			if err := dhcp.ListenAndServeIf(bridge, dhcpServer); err != nil {
				logger.Println(bridge+":", err)
			}
		}(bridge)
	}
	return dhcpServer, nil
}

func getMyIP() (net.IP, error) {
	var myIP net.IP
	mostOnesInMask := 0
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		if iface.Flags&net.FlagBroadcast == 0 {
			continue
		}
		interfaceAddrs, err := iface.Addrs()
		if err != nil {
			return nil, err
		}
		for _, addr := range interfaceAddrs {
			IP, IPNet, err := net.ParseCIDR(addr.String())
			if err != nil {
				return nil, err
			}
			if IP = IP.To4(); IP == nil {
				continue
			}
			if onesInMask, _ := IPNet.Mask.Size(); onesInMask > mostOnesInMask {
				myIP = IP
				mostOnesInMask = onesInMask
			}
		}
	}
	if myIP == nil {
		return nil, errors.New("no IP address found")
	}
	return myIP, nil
}

func findRoutes() ([]routeType, string, error) {
	file, err := os.Open("/proc/net/route")
	if err != nil {
		return nil, "", err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	routes := make([]routeType, 0)
	var defaultInterfaceName string
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
		if flags&0x1 != 0x1 {
			continue
		}
		routes = append(routes, routeType{interfaceName, destAddr, mask})
		if destAddr == 0 && flags&0x2 == 0x2 {
			defaultInterfaceName = interfaceName
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, "", err
	}
	return routes, defaultInterfaceName, nil
}

func (s *DhcpServer) acknowledgeLease(ipAddr net.IP) {
	ipStr := ipAddr.String()
	s.mutex.Lock()
	ackChan, ok := s.ackChannels[ipStr]
	delete(s.ackChannels, ipStr)
	s.mutex.Unlock()
	if ok {
		ackChan <- struct{}{}
		close(ackChan)
	}
}

func (s *DhcpServer) addLease(address proto.Address) {
	address.Shrink()
	ipAddr := address.IpAddress.String()
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.ipAddrToMacAddr[ipAddr] = address.MacAddress
	s.leases[address.MacAddress] = address
}

func (s *DhcpServer) addSubnet(subnet proto.Subnet) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.subnets = append(s.subnets, subnet)
}

func (s *DhcpServer) findLease(macAddr string) (*proto.Address, *proto.Subnet) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if address, ok := s.leases[macAddr]; !ok {
		return nil, nil
	} else {
		for _, subnet := range s.subnets {
			subnetMask := net.IPMask(subnet.IpMask)
			subnetAddr := subnet.IpGateway.Mask(subnetMask)
			if address.IpAddress.Mask(subnetMask).Equal(subnetAddr) {
				return &address, &subnet
			}
		}
		return &address, nil
	}
}

func (s *DhcpServer) makeAcknowledgmentChannel(ipAddr net.IP) <-chan struct{} {
	ipStr := ipAddr.String()
	newChan := make(chan struct{}, 1)
	s.mutex.Lock()
	oldChan, ok := s.ackChannels[ipStr]
	s.ackChannels[ipStr] = newChan
	s.mutex.Unlock()
	if ok {
		close(oldChan)
	}
	return newChan
}

func (s *DhcpServer) removeLease(ipAddr net.IP) {
	ipStr := ipAddr.String()
	s.mutex.Lock()
	defer s.mutex.Unlock()
	delete(s.leases, s.ipAddrToMacAddr[ipStr])
	delete(s.ipAddrToMacAddr, ipStr)
}

func (s *DhcpServer) ServeDHCP(req dhcp.Packet, msgType dhcp.MessageType,
	options dhcp.Options) dhcp.Packet {
	switch msgType {
	case dhcp.Discover:
		macAddr := req.CHAddr().String()
		s.logger.Debugf(1, "DHCP Discover from: %s\n", macAddr)
		lease, subnet := s.findLease(macAddr)
		if lease == nil {
			return nil
		}
		if subnet == nil {
			s.logger.Printf("No subnet found for %s\n", lease.IpAddress)
			return nil
		}
		s.logger.Debugf(0, "DHCP Offer: %s for: %s, server: %s\n",
			lease.IpAddress, macAddr, s.myIP)
		dnsServers := make([]byte, 0)
		for _, dnsServer := range subnet.DomainNameServers {
			dnsServers = append(dnsServers, dnsServer...)
		}
		leaseOptions := dhcp.Options{
			dhcp.OptionSubnetMask:       subnet.IpMask,
			dhcp.OptionRouter:           subnet.IpGateway,
			dhcp.OptionDomainNameServer: dnsServers,
		}
		return dhcp.ReplyPacket(req, dhcp.Offer, s.myIP, lease.IpAddress,
			leaseTime,
			leaseOptions.SelectOrderOrAll(
				options[dhcp.OptionParameterRequestList]))
	case dhcp.Request:
		server, ok := options[dhcp.OptionServerIdentifier]
		if ok {
			serverIP := net.IP(server)
			if !serverIP.IsUnspecified() && !serverIP.Equal(s.myIP) {
				s.logger.Debugf(0, "DHCP Request to: %s is not me: %s\n",
					serverIP, s.myIP)
				return nil // Message not for this DHCP server.
			}
		}
		reqIP := net.IP(options[dhcp.OptionRequestedIPAddress])
		if reqIP == nil {
			s.logger.Debugln(0, "DHCP Request did not request an IP")
			reqIP = net.IP(req.CIAddr())
		}
		macAddr := req.CHAddr().String()
		s.logger.Debugf(0, "DHCP Request for: %s from: %s\n", reqIP, macAddr)
		lease, subnet := s.findLease(macAddr)
		if lease == nil {
			s.logger.Printf("No lease found for %s\n", macAddr)
			return nil
		}
		if subnet == nil {
			s.logger.Printf("No subnet found for %s\n", lease.IpAddress)
			return nil
		}
		if reqIP.Equal(lease.IpAddress) {
			dnsServers := make([]byte, 0)
			for _, dnsServer := range subnet.DomainNameServers {
				dnsServers = append(dnsServers, dnsServer...)
			}
			leaseOptions := dhcp.Options{
				dhcp.OptionSubnetMask:       subnet.IpMask,
				dhcp.OptionRouter:           subnet.IpGateway,
				dhcp.OptionDomainNameServer: dnsServers,
			}
			s.logger.Debugf(0, "DHCP ACK for: %s to: %s\n", reqIP, macAddr)
			s.acknowledgeLease(lease.IpAddress)
			return dhcp.ReplyPacket(req, dhcp.ACK, s.myIP, reqIP, leaseTime,
				leaseOptions.SelectOrderOrAll(
					options[dhcp.OptionParameterRequestList]))
		} else {
			s.logger.Debugf(0, "DHCP NAK for: %s to: %s\n", reqIP, macAddr)
			return dhcp.ReplyPacket(req, dhcp.NAK, s.myIP, nil, 0, nil)
		}
	default:
		s.logger.Debugf(0, "Unsupported message type: %s\n", msgType)
	}
	return nil
}