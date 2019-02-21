package dhcpd

import (
	"errors"
	"net"
	"strings"
	"time"

	"github.com/Symantec/Dominator/lib/log"
	libnet "github.com/Symantec/Dominator/lib/net"
	"github.com/Symantec/Dominator/lib/net/util"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
	dhcp "github.com/krolaw/dhcp4"
	"golang.org/x/net/ipv4"
)

const sysClassNet = "/sys/class/net"
const leaseTime = time.Hour * 48

type serveIfConn struct {
	ifIndices map[int]struct{}
	conn      *ipv4.PacketConn
	cm        *ipv4.ControlMessage
}

func newServer(interfaceNames []string, logger log.DebugLogger) (
	*DhcpServer, error) {
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
	if len(interfaceNames) < 1 {
		logger.Debugln(0, "Starting DHCP server on all broadcast interfaces")
		interfaces, _, err := libnet.ListBroadcastInterfaces(
			libnet.InterfaceTypeEtherNet|libnet.InterfaceTypeBridge, logger)
		if err != nil {
			return nil, err
		} else {
			for _, iface := range interfaces {
				interfaceNames = append(interfaceNames, iface.Name)
			}
		}
	} else {
		logger.Debugln(0, "Starting DHCP server on interfaces: "+
			strings.Join(interfaceNames, ","))
	}
	serveConn := &serveIfConn{
		ifIndices: make(map[int]struct{}, len(interfaceNames)),
	}
	for _, interfaceName := range interfaceNames {
		if iface, err := net.InterfaceByName(interfaceName); err != nil {
			return nil, err
		} else {
			serveConn.ifIndices[iface.Index] = struct{}{}
		}
	}
	listener, err := net.ListenPacket("udp4", ":67")
	if err != nil {
		return nil, err
	}
	pktConn := ipv4.NewPacketConn(listener)
	if err := pktConn.SetControlMessage(ipv4.FlagInterface, true); err != nil {
		listener.Close()
		return nil, err
	}
	serveConn.conn = pktConn
	go func() {
		if err := dhcp.Serve(serveConn, dhcpServer); err != nil {
			logger.Println(err)
		}
	}()
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

func (s *DhcpServer) addLease(address proto.Address, doNetboot bool,
	hostname string, subnet *proto.Subnet) error {
	address.Shrink()
	if len(address.IpAddress) < 1 {
		return errors.New("no IP address")
	}
	ipAddr := address.IpAddress.String()
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if subnet == nil {
		if subnet = s.findMatchingSubnet(address.IpAddress); subnet == nil {
			return errors.New("no subnet found for " + ipAddr)
		}
	}
	if doNetboot {
		if len(s.networkBootImage) < 1 {
			return errors.New("no Network Boot Image name configured")
		}
		if _, ok := s.leases[address.MacAddress]; ok {
			return errors.New("already have lease for: " + address.MacAddress)
		}
	}
	s.ipAddrToMacAddr[ipAddr] = address.MacAddress
	s.leases[address.MacAddress] = leaseType{
		address, hostname, doNetboot, subnet}
	return nil
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
	} else if lease.subnet != nil {
		return &lease, lease.subnet
	} else {
		return &lease, s.findMatchingSubnet(lease.IpAddress)
	}
}

// This must be called with the lock held.
func (s *DhcpServer) findMatchingSubnet(ipAddr net.IP) *proto.Subnet {
	for _, subnet := range s.subnets {
		subnetMask := net.IPMask(subnet.IpMask)
		subnetAddr := subnet.IpGateway.Mask(subnetMask)
		if ipAddr.Mask(subnetMask).Equal(subnetAddr) {
			return &subnet
		}
	}
	return nil
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

func (s *DhcpServer) makeOptions(subnet *proto.Subnet,
	lease *leaseType) dhcp.Options {
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
	if lease.doNetboot {
		leaseOptions[dhcp.OptionTFTPServerName] = s.myIP
		leaseOptions[dhcp.OptionBootFileName] = s.networkBootImage
	}
	return leaseOptions
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
	delete(s.leases, s.ipAddrToMacAddr[ipStr])
	delete(s.ipAddrToMacAddr, ipStr)
	ackChan, ok := s.ackChannels[ipStr]
	delete(s.ackChannels, ipStr)
	s.mutex.Unlock()
	if ok {
		close(ackChan)
	}
}

func (s *DhcpServer) removeSubnet(subnetId string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	subnets := make([]proto.Subnet, 0, len(s.subnets)-1)
	for _, subnet := range s.subnets {
		if subnet.Id != subnetId {
			subnets = append(subnets, subnet)
		}
	}
	s.subnets = subnets
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
		leaseOptions := s.makeOptions(subnet, lease)
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
			leaseOptions := s.makeOptions(subnet, lease)
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

func (s *serveIfConn) ReadFrom(b []byte) (n int, addr net.Addr, err error) {
	for {
		n, s.cm, addr, err = s.conn.ReadFrom(b)
		if err != nil || s.cm == nil {
			break
		}
		if _, ok := s.ifIndices[s.cm.IfIndex]; ok {
			break
		}
	}
	return
}

func (s *serveIfConn) WriteTo(b []byte, addr net.Addr) (n int, err error) {
	s.cm.Src = nil
	return s.conn.WriteTo(b, s.cm, addr)
}
