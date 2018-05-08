package manager

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path"

	"github.com/Symantec/Dominator/lib/json"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
)

func (m *Manager) addAddressesToPool(addresses []proto.Address,
	lock bool) error {
	for index := range addresses {
		addresses[index].Shrink()
	}
	if lock {
		m.mutex.Lock()
		defer m.mutex.Unlock()
	}
	existingIpAddresses := make(map[string]struct{})
	existingMacAddresses := make(map[string]struct{})
	for _, address := range m.addressPool.Registered {
		if address.IpAddress != nil {
			existingIpAddresses[address.IpAddress.String()] = struct{}{}
		}
		existingMacAddresses[address.MacAddress] = struct{}{}
	}
	for ipAddress, vm := range m.vms {
		existingIpAddresses[ipAddress] = struct{}{}
		existingMacAddresses[vm.Address.MacAddress] = struct{}{}
	}
	for _, address := range addresses {
		ipAddr := address.IpAddress
		if ipAddr != nil {
			if m.getMatchingSubnet(ipAddr) == "" {
				return fmt.Errorf("no subnet matching: %s", address.IpAddress)
			}
			if _, ok := existingIpAddresses[ipAddr.String()]; ok {
				return fmt.Errorf("duplicate IP address: %s", address.IpAddress)
			}
		}
		if _, ok := existingMacAddresses[address.MacAddress]; ok {
			return fmt.Errorf("duplicate MAC address: %s", address.MacAddress)
		}
	}
	m.addressPool.Free = append(m.addressPool.Free, addresses...)
	m.addressPool.Registered = append(m.addressPool.Registered, addresses...)
	return m.writeAddressPool(m.addressPool, true)
}

func (m *Manager) loadAddressPool() error {
	var addressPool addressPoolType
	err := json.ReadFromFile(path.Join(m.StateDir, "address-pool.json"),
		&addressPool)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	for index := range addressPool.Free {
		addressPool.Free[index].Shrink()
	}
	for index := range addressPool.Registered {
		addressPool.Registered[index].Shrink()
	}
	m.addressPool = addressPool
	return nil
}

func (m *Manager) getFreeAddress(subnetId string) (
	proto.Address, string, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if len(m.addressPool.Free) < 1 {
		return proto.Address{}, "", errors.New("no free addresses in pool")
	}
	if subnetId == "" {
		address := m.addressPool.Free[0]
		subnetId = m.getMatchingSubnet(address.IpAddress)
		if subnetId == "" {
			return proto.Address{}, "",
				fmt.Errorf("no subnet matching: %s", address.IpAddress)
		}
		newPool := addressPoolType{
			Free:       make([]proto.Address, len(m.addressPool.Free)-1),
			Registered: m.addressPool.Registered,
		}
		copy(newPool.Free, m.addressPool.Free[1:])
		if err := m.writeAddressPool(newPool, false); err != nil {
			return proto.Address{}, "", err
		}
		m.addressPool = newPool
		return address, subnetId, nil
	}
	subnet, ok := m.subnets[subnetId]
	if !ok {
		return proto.Address{}, "", fmt.Errorf("no such subnet: %s", subnetId)
	}
	subnetMask := net.IPMask(subnet.IpMask)
	subnetAddr := subnet.IpGateway.Mask(subnetMask)
	foundPos := -1
	for index, address := range m.addressPool.Free {
		if address.IpAddress.Mask(subnetMask).Equal(subnetAddr) {
			foundPos = index
			break
		}
	}
	if foundPos < 0 {
		return proto.Address{}, "",
			fmt.Errorf("no free address in subnet: %s", subnetId)
	}
	addressPool := addressPoolType{
		Free:       make([]proto.Address, 0, len(m.addressPool.Free)-1),
		Registered: m.addressPool.Registered,
	}
	for index, address := range m.addressPool.Free {
		if index == foundPos {
			break
		}
		addressPool.Free = append(addressPool.Free, address)
	}
	if err := m.writeAddressPool(addressPool, false); err != nil {
		return proto.Address{}, "", err
	}
	address := m.addressPool.Free[foundPos]
	m.addressPool = addressPool
	return address, subnetId, nil
}

func (m *Manager) listAvailableAddresses() []proto.Address {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	addresses := make([]proto.Address, 0, len(m.addressPool.Free))
	for _, address := range m.addressPool.Free {
		addresses = append(addresses, address)
	}
	return addresses
}

func (m *Manager) releaseAddressInPool(address proto.Address, lock bool) error {
	if lock {
		m.mutex.Lock()
		defer m.mutex.Unlock()
	}
	m.addressPool.Free = append(m.addressPool.Free, address)
	return m.writeAddressPool(m.addressPool, false)
}

func (m *Manager) removeExcessAddressesFromPool(maxFree uint) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if uint(len(m.addressPool.Free)) <= maxFree {
		return nil
	}
	newFree := make([]proto.Address, maxFree)
	copy(newFree, m.addressPool.Free)
	macAddressesToRemove := make(map[string]struct{})
	for _, address := range m.addressPool.Free[maxFree:] {
		macAddressesToRemove[address.MacAddress] = struct{}{}
	}
	newRegistered := make([]proto.Address, 0,
		len(m.addressPool.Registered)-len(macAddressesToRemove))
	for _, address := range m.addressPool.Registered {
		if _, ok := macAddressesToRemove[address.MacAddress]; !ok {
			newRegistered = append(newRegistered, address)
		}
	}
	newPool := addressPoolType{newFree, newRegistered}
	if err := m.writeAddressPool(newPool, true); err != nil {
		return err
	}
	m.addressPool = newPool
	return nil
}

func (m *Manager) writeAddressPool(addressPool addressPoolType,
	sendAll bool) error {
	err := json.WriteToFile(path.Join(m.StateDir, "address-pool.json"),
		publicFilePerms, "    ", addressPool)
	if err != nil {
		return err
	}
	update := proto.Update{
		HaveNumFree:      true,
		NumFreeAddresses: uint(len(addressPool.Free)),
	}
	if sendAll {
		update.HaveAddressPool = true
		update.AddressPool = addressPool.Registered
	}
	m.sendUpdateWithLock(update)
	return nil
}
