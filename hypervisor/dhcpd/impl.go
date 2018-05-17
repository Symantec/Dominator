package dhcpd

import (
	"net"
	"time"

	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/net/util"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
	dhcp "github.com/krolaw/dhcp4"
)

const sysClassNet = "/sys/class/net"
const leaseTime = time.Hour * 48

func makeOptions(subnet *proto.Subnet, lease *leaseType) dhcp.Options {
	dnsServers := make([]byte, 0)
	for _, dnsServer := range subnet.DomainNameServers {
		dnsServers = append(dnsServers, dnsServer...)
	}
	leaseOptions := dhcp.Options{
		dhcp.OptionSubnetMask:       subnet.IpMask,
		dhcp.OptionRouter:           subnet.IpGateway,
		dhcp.OptionDomainNameServer: dnsServers,
	}
	if subnet.DomainName != "" {
		leaseOptions[dhcp.OptionDomainName] = []byte(subnet.DomainName)
	}
	if lease.Hostname != "" {
		leaseOptions[dhcp.OptionHostName] = []byte(lease.Hostname)
	}
	return leaseOptions
}

func newServer(bridges []string, logger log.DebugLogger) (*DhcpServer, error) {
	dhcpServer := &DhcpServer{
		logger:          logger,
		ackChannels:     make(map[string]chan struct{}),
		ipAddrToMacAddr: make(map[string]string),
		leases:          make(map[string]leaseType),
		requestChannels: make(map[string]chan net.IP),
	}
	if myIP, err := util.GetMyIP(); err != nil {
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

func (s *DhcpServer) addLease(address proto.Address, hostname string) {
	address.Shrink()
	if len(address.IpAddress) < 1 {
		return
	}
	ipAddr := address.IpAddress.String()
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.ipAddrToMacAddr[ipAddr] = address.MacAddress
	s.leases[address.MacAddress] = leaseType{address, hostname}
}

func (s *DhcpServer) addSubnet(subnet proto.Subnet) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.subnets = append(s.subnets, subnet)
}

func (s *DhcpServer) findLease(macAddr string) (*leaseType, *proto.Subnet) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if lease, ok := s.leases[macAddr]; !ok {
		return nil, nil
	} else {
		for _, subnet := range s.subnets {
			subnetMask := net.IPMask(subnet.IpMask)
			subnetAddr := subnet.IpGateway.Mask(subnetMask)
			if lease.IpAddress.Mask(subnetMask).Equal(subnetAddr) {
				return &lease, &subnet
			}
		}
		return &lease, nil
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

func (s *DhcpServer) makeRequestChannel(macAddr string) <-chan net.IP {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if oldChan, ok := s.requestChannels[macAddr]; ok {
		return oldChan
	}
	newChan := make(chan net.IP, 16)
	s.requestChannels[macAddr] = newChan
	return newChan
}

func (s *DhcpServer) notifyRequest(address proto.Address) {
	s.mutex.Lock()
	requestChan, ok := s.requestChannels[address.MacAddress]
	s.mutex.Unlock()
	if ok {
		select {
		case requestChan <- address.IpAddress:
		default:
		}
	}
}

func (s *DhcpServer) removeLease(ipAddr net.IP) {
	if len(ipAddr) < 1 {
		return
	}
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
		leaseOptions := makeOptions(subnet, lease)
		return dhcp.ReplyPacket(req, dhcp.Offer, s.myIP, lease.IpAddress,
			leaseTime,
			leaseOptions.SelectOrderOrAll(
				options[dhcp.OptionParameterRequestList]))
	case dhcp.Request:
		reqIP := net.IP(options[dhcp.OptionRequestedIPAddress])
		if reqIP == nil {
			s.logger.Debugln(0, "DHCP Request did not request an IP")
			reqIP = net.IP(req.CIAddr())
		}
		reqIP = util.ShrinkIP(reqIP)
		macAddr := req.CHAddr().String()
		s.notifyRequest(proto.Address{reqIP, macAddr})
		server, ok := options[dhcp.OptionServerIdentifier]
		if ok {
			serverIP := net.IP(server)
			if !serverIP.IsUnspecified() && !serverIP.Equal(s.myIP) {
				s.logger.Debugf(0,
					"DHCP Request for: %s from: %s to: %s is not me: %s\n",
					reqIP, macAddr, serverIP, s.myIP)
				return nil // Message not for this DHCP server.
			}
		}
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
			leaseOptions := makeOptions(subnet, lease)
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
