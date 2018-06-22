package manager

import (
	"fmt"
	"net"
	"os"
	"path"
	"sort"

	"github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/lib/net/util"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
)

func getHypervisorSubnet() (proto.Subnet, error) {
	defaultRoute, err := util.GetDefaultRoute()
	if err != nil {
		return proto.Subnet{}, err
	}
	resolverConfig, err := util.GetResolverConfiguration()
	if err != nil {
		return proto.Subnet{}, err
	}
	myIP, err := util.GetMyIP()
	if err != nil {
		return proto.Subnet{}, err
	}
	nameservers := make([]net.IP, 0, len(resolverConfig.Nameservers))
	for _, nameserver := range resolverConfig.Nameservers {
		if nameserver[0] == 127 {
			nameservers = append(nameservers, myIP)
		} else {
			nameservers = append(nameservers, nameserver)
		}
	}
	return proto.Subnet{
		Id:                "hypervisor",
		IpGateway:         defaultRoute.Address,
		IpMask:            net.IP(defaultRoute.Mask),
		DomainNameServers: nameservers,
	}, nil
}

func (m *Manager) updateSubnets(request proto.UpdateSubnetsRequest) error {
	for index, subnet := range request.Add {
		if subnet.Id == "hypervisor" {
			return fmt.Errorf("cannot add hypervisor subnet")
		}
		request.Add[index].Shrink()
	}
	for index, subnet := range request.Change {
		if subnet.Id == "hypervisor" {
			return fmt.Errorf("cannot change hypervisor subnet")
		}
		request.Change[index].Shrink()
	}
	for _, subnetId := range request.Delete {
		if subnetId == "hypervisor" {
			return fmt.Errorf("cannot delete hypervisor subnet")
		}
	}
	if err := m.updateSubnetsLocked(request); err != nil {
		return err
	}
	for _, subnet := range request.Add {
		m.DhcpServer.AddSubnet(subnet)
		for _, ch := range m.subnetChannels {
			ch <- subnet
		}
	}
	for _, subnet := range request.Change {
		m.DhcpServer.RemoveSubnet(subnet.Id)
		m.DhcpServer.AddSubnet(subnet)
		// TOOO(rgooch): Design a clean way to send updates to the channels.
	}
	for _, subnetId := range request.Delete {
		m.DhcpServer.RemoveSubnet(subnetId)
		// TOOO(rgooch): Design a clean way to send deletes to the channels.
	}
	return nil
}

func (m *Manager) updateSubnetsLocked(
	request proto.UpdateSubnetsRequest) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	for _, subnet := range request.Add {
		if _, ok := m.subnets[subnet.Id]; ok {
			return fmt.Errorf("subnet: %s already exists", subnet.Id)
		}
	}
	for _, subnet := range request.Change {
		if _, ok := m.subnets[subnet.Id]; !ok {
			return fmt.Errorf("subnet: %s does not exist", subnet.Id)
		}
	}
	for _, subnetId := range request.Delete {
		if _, ok := m.subnets[subnetId]; !ok {
			return fmt.Errorf("subnet: %s does not exist", subnetId)
		}
	}
	for _, subnet := range request.Add {
		m.subnets[subnet.Id] = subnet
	}
	for _, subnet := range request.Change {
		m.subnets[subnet.Id] = subnet
	}
	for _, subnetId := range request.Delete {
		delete(m.subnets, subnetId)
	}
	subnetsToWrite := make([]proto.Subnet, 0, len(m.subnets)-1)
	for _, subnet := range m.subnets {
		if subnet.Id != "hypervisor" {
			subnetsToWrite = append(subnetsToWrite, subnet)
		}
	}
	err := json.WriteToFile(path.Join(m.StateDir, "subnets.json"),
		publicFilePerms, "    ", subnetsToWrite)
	if err != nil {
		return err
	}
	m.sendUpdateWithLock(
		proto.Update{HaveSubnets: true, Subnets: subnetsToWrite})
	return nil
}

// This must be called with the lock held.
func (m *Manager) getMatchingSubnet(ipAddr net.IP) string {
	for id, subnet := range m.subnets {
		subnetMask := net.IPMask(subnet.IpMask)
		subnetAddr := subnet.IpGateway.Mask(subnetMask)
		if ipAddr.Mask(subnetMask).Equal(subnetAddr) {
			return id
		}
	}
	return ""
}

func (m *Manager) listSubnets(doSort bool) []proto.Subnet {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	subnets := make([]proto.Subnet, 0, len(m.subnets))
	if !doSort {
		for _, subnet := range m.subnets {
			subnets = append(subnets, subnet)
		}
		return subnets
	}
	subnetIDs := make([]string, 0, len(m.subnets))
	for subnetID := range m.subnets {
		subnetIDs = append(subnetIDs, subnetID)
	}
	sort.Strings(subnetIDs)
	for _, subnetID := range subnetIDs {
		subnets = append(subnets, m.subnets[subnetID])
	}
	return subnets
}

// This returns with the Manager locked, waiting for existing subnets to be
// drained from the channel by the caller before unlocking.
func (m *Manager) makeSubnetChannel() <-chan proto.Subnet {
	ch := make(chan proto.Subnet, 1)
	m.mutex.Lock()
	m.subnetChannels = append(m.subnetChannels, ch)
	go func() {
		defer m.mutex.Unlock()
		for _, subnet := range m.subnets {
			ch <- subnet
		}
	}()
	return ch
}

func (m *Manager) loadSubnets() error {
	var subnets []proto.Subnet
	err := json.ReadFromFile(path.Join(m.StateDir, "subnets.json"), &subnets)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	for index := range subnets {
		subnets[index].Shrink()
		m.DhcpServer.AddSubnet(subnets[index])
	}
	m.subnets = make(map[string]proto.Subnet, len(subnets))
	for _, subnet := range subnets {
		m.subnets[subnet.Id] = subnet
	}
	if subnet, err := getHypervisorSubnet(); err != nil {
		return err
	} else {
		m.subnets["hypervisor"] = subnet
	}
	return nil
}
