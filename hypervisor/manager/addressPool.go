package manager

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path"

	"github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
)

func ipIsUnspecified(ipAddr net.IP) bool {
	if len(ipAddr) < 1 {
		return true
	}
	return ipAddr.IsUnspecified()
}

func (m *Manager) addAddressesToPool(addresses []proto.Address) error {
	for index := range addresses {
		addresses[index].Shrink()
	}
	existingIpAddresses := make(map[string]struct{})
	existingMacAddresses := make(map[string]struct{})
	m.mutex.Lock()
	defer m.mutex.Unlock()
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
	m.Logger.Debugf(0, "adding %d addresses to pool\n", len(addresses))
	m.addressPool.Free = append(m.addressPool.Free, addresses...)
	m.addressPool.Registered = append(m.addressPool.Registered, addresses...)
	return m.writeAddressPoolWithLock(m.addressPool, true)
}

func (m *Manager) computeNumFreeAddressesMap(addressPool addressPoolType) (
	map[string]uint, error) {
	numFreeAddresses := make(map[string]uint, len(m.subnets))
	for subnetId := range m.subnets {
		if subnetId == "" || subnetId == "hypervisor" {
			continue
		}
		numFreeAddresses[subnetId] = 0
	}
	for _, address := range addressPool.Free {
		if len(address.IpAddress) < 1 {
			continue
		}
		if subnetId := m.getMatchingSubnet(address.IpAddress); subnetId == "" {
			return nil,
				fmt.Errorf("no matching subnet for: %s\n", address.IpAddress)
		} else if subnetId != "hypervisor" {
			numFreeAddresses[subnetId]++
		}
	}
	return numFreeAddresses, nil
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

func (m *Manager) getFreeAddress(ipAddr net.IP, subnetId string,
	authInfo *srpc.AuthInformation) (proto.Address, string, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if len(m.addressPool.Free) < 1 {
		return proto.Address{}, "", errors.New("no free addresses in pool")
	}
	if subnet, err := m.getSubnetAndAuth(subnetId, authInfo); err != nil {
		return proto.Address{}, "", err
	} else {
		subnetMask := net.IPMask(subnet.IpMask)
		subnetAddr := subnet.IpGateway.Mask(subnetMask)
		foundPos := -1
		for index, address := range m.addressPool.Free {
			if !ipIsUnspecified(ipAddr) && !ipAddr.Equal(address.IpAddress) {
				continue
			}
			if address.IpAddress.Mask(subnetMask).Equal(subnetAddr) {
				foundPos = index
				break
			}
		}
		if foundPos < 0 {
			if ipIsUnspecified(ipAddr) {
				return proto.Address{}, "",
					fmt.Errorf("no free address in subnet: %s", subnetId)
			} else {
				return proto.Address{}, "",
					fmt.Errorf("address: %s not found in free pool", ipAddr)
			}
		}
		addressPool := addressPoolType{
			Free:       make([]proto.Address, 0, len(m.addressPool.Free)-1),
			Registered: m.addressPool.Registered,
		}
		for index, address := range m.addressPool.Free {
			if index == foundPos {
				continue
			}
			addressPool.Free = append(addressPool.Free, address)
		}
		if err := m.writeAddressPoolWithLock(addressPool, false); err != nil {
			return proto.Address{}, "", err
		}
		address := m.addressPool.Free[foundPos]
		m.addressPool = addressPool
		return address, subnet.Id, nil
	}
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

func (m *Manager) registerAddress(address proto.Address) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.addressPool.Registered = append(m.addressPool.Registered, address)
	if err := m.writeAddressPoolWithLock(m.addressPool, true); err != nil {
		return err
	}
	return nil
}

func (m *Manager) releaseAddressInPool(address proto.Address) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.releaseAddressInPoolWithLock(address)
}

func (m *Manager) releaseAddressInPoolWithLock(address proto.Address) error {
	m.addressPool.Free = append(m.addressPool.Free, address)
	return m.writeAddressPoolWithLock(m.addressPool, false)
}

func (m *Manager) removeExcessAddressesFromPool(maxFree map[string]uint) error {
	freeCount := make(map[string]uint)
	macAddressesToRemove := make(map[string]struct{})
	m.mutex.Lock()
	defer m.mutex.Unlock()
	// TODO(rgooch): Should precompute a map to avoid this N*M loop.
	for _, address := range m.addressPool.Free {
		subnetId := m.getMatchingSubnet(address.IpAddress)
		freeCount[subnetId]++
		if maxFree, ok := maxFree[subnetId]; ok {
			if freeCount[subnetId] > maxFree {
				macAddressesToRemove[address.MacAddress] = struct{}{}
			}
		}
		if maxFree, ok := maxFree[""]; ok {
			if freeCount[subnetId] > maxFree {
				macAddressesToRemove[address.MacAddress] = struct{}{}
			}
		}
	}
	if len(macAddressesToRemove) < 1 {
		return nil
	}
	newPool := addressPoolType{}
	for _, address := range m.addressPool.Free {
		if _, ok := macAddressesToRemove[address.MacAddress]; !ok {
			newPool.Free = append(newPool.Free, address)
		}
	}
	for _, address := range m.addressPool.Registered {
		if _, ok := macAddressesToRemove[address.MacAddress]; !ok {
			newPool.Registered = append(newPool.Registered, address)
		}
	}
	m.Logger.Debugf(0,
		"removing %d addresses from pool, leaving %d with %d free\n",
		len(macAddressesToRemove), len(newPool.Registered), len(newPool.Free))
	if err := m.writeAddressPoolWithLock(newPool, true); err != nil {
		return err
	}
	m.addressPool = newPool
	return nil
}

func (m *Manager) unregisterAddress(address proto.Address) error {
	found := false
	m.mutex.Lock()
	defer m.mutex.Unlock()
	addresses := make([]proto.Address, 0, len(m.addressPool.Registered)-1)
	for _, addr := range m.addressPool.Registered {
		if address.Equal(&addr) {
			found = true
		} else {
			addresses = append(addresses, addr)
		}
	}
	if !found {
		return fmt.Errorf("%v not registered", address)
	}
	m.addressPool.Registered = addresses
	return m.writeAddressPoolWithLock(m.addressPool, true)
}

func (m *Manager) writeAddressPool(addressPool addressPoolType,
	sendAll bool) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.writeAddressPoolWithLock(addressPool, sendAll)
}

func (m *Manager) writeAddressPoolWithLock(addressPool addressPoolType,
	sendAll bool) error {
	// TODO(rgooch): Should precompute the numFreeAddresses map.
	numFreeAddresses, err := m.computeNumFreeAddressesMap(addressPool)
	if err != nil {
		return err
	}
	err = json.WriteToFile(path.Join(m.StateDir, "address-pool.json"),
		publicFilePerms, "    ", addressPool)
	if err != nil {
		return err
	}
	update := proto.Update{NumFreeAddresses: numFreeAddresses}
	if sendAll {
		update.HaveAddressPool = true
		update.AddressPool = addressPool.Registered
	}
	m.sendUpdateWithLock(update)
	return nil
}
