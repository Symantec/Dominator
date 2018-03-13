package manager

import (
	"fmt"
	"os"
	"path"

	"github.com/Symantec/Dominator/lib/json"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
)

func (m *Manager) addSubnets(subnets []proto.Subnet) error {
	if err := m.addSubnetsInternal(subnets); err != nil {
		return err
	}
	for _, subnet := range subnets {
		m.DhcpServer.AddSubnet(subnet)
		for _, ch := range m.subnetChannels {
			ch <- subnet
		}
	}
	return nil
}

func (m *Manager) addSubnetsInternal(subnets []proto.Subnet) error {
	for _, subnet := range subnets {
		subnet.Shrink()
	}
	m.mutex.Lock()
	defer m.mutex.Unlock()
	for _, subnet := range subnets {
		if _, ok := m.subnets[subnet.Id]; ok {
			return fmt.Errorf("subnet: %s already exists", subnet.Id)
		}
	}
	for _, subnet := range subnets {
		m.subnets[subnet.Id] = subnet
	}
	return json.WriteToFile(path.Join(m.StateDir, "subnets.json"),
		filePerms, "    ", m.subnets)
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
	return nil
}
