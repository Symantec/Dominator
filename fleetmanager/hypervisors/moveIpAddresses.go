package hypervisors

import (
	"fmt"
	"net"
	"time"

	"github.com/Cloud-Foundations/Dominator/lib/constants"
	"github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/net/util"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	hyper_proto "github.com/Cloud-Foundations/Dominator/proto/hypervisor"
)

func (m *Manager) addIp(hypervisorIpAddress, ip net.IP) error {
	client, err := srpc.DialHTTP("tcp",
		fmt.Sprintf("%s:%d",
			hypervisorIpAddress, constants.HypervisorPortNumber),
		time.Second*15)
	if err != nil {
		return err
	}
	defer client.Close()
	request := hyper_proto.ChangeAddressPoolRequest{
		AddressesToAdd: []hyper_proto.Address{{
			IpAddress: ip,
			MacAddress: fmt.Sprintf("52:54:%02x:%02x:%02x:%02x",
				ip[0], ip[1], ip[2], ip[3]),
		}},
	}
	var reply hyper_proto.ChangeAddressPoolResponse
	err = client.RequestReply("Hypervisor.ChangeAddressPool", request, &reply)
	if err != nil {
		return err
	}
	return errors.New(reply.Error)
}

func (m *Manager) getHealthyHypervisorAddr(hostname string) (net.IP, error) {
	hypervisor, err := m.getLockedHypervisor(hostname, false)
	if err != nil {
		return nil, err
	}
	defer hypervisor.mutex.RUnlock()
	if hypervisor.healthStatus == "marginal" ||
		hypervisor.healthStatus == "at risk" {
		return nil, errors.New("cannot move IPs to unhealthy hypervisor")
	}
	if len(hypervisor.machine.HostIpAddress) < 1 {
		return nil, fmt.Errorf("IP address for: %s not known", hostname)
	}
	return hypervisor.machine.HostIpAddress, nil
}

func (m *Manager) markIPsForMigration(ipAddresses []net.IP) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if num := len(m.migratingIPs); num > 0 {
		return fmt.Errorf("%d other migrations in progress: %v",
			num, m.migratingIPs)
	}
	for _, ip := range ipAddresses {
		if len(ip) > 0 {
			m.migratingIPs[ip.String()] = struct{}{}
		}
	}
	return nil
}

func (m *Manager) moveIpAddresses(hostname string, ipAddresses []net.IP) error {
	if !*manageHypervisors {
		return errors.New("this is a read-only Fleet Manager")
	}
	if len(ipAddresses) < 1 {
		return nil
	}
	sourceHypervisorIPs := make([]net.IP, len(ipAddresses))
	for index, ip := range ipAddresses {
		ip = util.ShrinkIP(ip)
		ipAddresses[index] = ip
		sourceHypervisorIp, err := m.storer.GetHypervisorForIp(ip)
		if err != nil {
			return err
		}
		sourceHypervisorIPs[index] = sourceHypervisorIp
	}
	hypervisorIpAddress, err := m.getHealthyHypervisorAddr(hostname)
	if err != nil {
		return err
	}
	if err := m.markIPsForMigration(ipAddresses); err != nil {
		return err
	}
	defer m.unmarkIPsForMigration(ipAddresses)
	// Move IPs.
	for index, ip := range ipAddresses {
		err := m.moveIpAddress(hypervisorIpAddress, sourceHypervisorIPs[index],
			ip)
		if err != nil {
			return err
		}
	}
	// Wait for IPs to have moved.
	// TODO(rgooch): Change this to watch for the registration events.
	stopTime := time.Now().Add(time.Second * 10)
	for ; time.Until(stopTime) >= 0; time.Sleep(time.Millisecond * 10) {
		allInPlace := true
		for _, ip := range ipAddresses {
			newHyperIp, err := m.storer.GetHypervisorForIp(ip)
			if err != nil {
				return err
			}
			if newHyperIp == nil || !newHyperIp.Equal(hypervisorIpAddress) {
				allInPlace = false
				break // Not yet registered with the destination Hypervisor.
			}
		}
		if allInPlace {
			return nil
		}
	}
	return errors.New("timed out waiting for addresses to move")
}

func (m *Manager) moveIpAddress(destinationHypervisorIpAddress,
	sourceHypervisorIpAddress, ipToMove net.IP) error {
	if sourceHypervisorIpAddress != nil {
		if sourceHypervisorIpAddress.Equal(destinationHypervisorIpAddress) {
			return nil // IP address is already registered to dest Hypervisor.
		}
		err := m.removeIpAndWait(sourceHypervisorIpAddress, ipToMove)
		if err != nil {
			return err
		}
	}
	return m.addIp(destinationHypervisorIpAddress, ipToMove)
}

func (m *Manager) removeIpAndWait(hypervisorIpAddress, ipToMove net.IP) error {
	client, err := srpc.DialHTTP("tcp",
		fmt.Sprintf("%s:%d",
			hypervisorIpAddress, constants.HypervisorPortNumber),
		time.Second*15)
	if err != nil {
		return err
	}
	defer client.Close()
	request := hyper_proto.ChangeAddressPoolRequest{
		AddressesToRemove: []hyper_proto.Address{{IpAddress: ipToMove}},
	}
	var reply hyper_proto.ChangeAddressPoolResponse
	err = client.RequestReply("Hypervisor.ChangeAddressPool", request, &reply)
	if err != nil {
		return err
	}
	if err := errors.New(reply.Error); err != nil {
		return fmt.Errorf("error unregistering %s from %s: %s",
			ipToMove, hypervisorIpAddress, err)
	}
	// TODO(rgooch): Change this to watch for the deregistration event.
	stopTime := time.Now().Add(time.Second * 10)
	for ; time.Until(stopTime) >= 0; time.Sleep(time.Millisecond * 10) {
		newHyperIp, err := m.storer.GetHypervisorForIp(ipToMove)
		if err != nil {
			return err
		}
		if newHyperIp == nil {
			return nil // No longer registered with a Hypervisor.
		}
	}
	return fmt.Errorf(
		"timed out waiting for %s to become unregistered from %s",
		ipToMove, hypervisorIpAddress)
}

func (m *Manager) unmarkIPsForMigration(ipAddresses []net.IP) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	for _, ip := range ipAddresses {
		if len(ip) > 0 {
			delete(m.migratingIPs, ip.String())
		}
	}
}
