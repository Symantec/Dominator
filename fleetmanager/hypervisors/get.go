package hypervisors

import (
	"errors"
	"net"

	"github.com/Cloud-Foundations/Dominator/fleetmanager/topology"
	proto "github.com/Cloud-Foundations/Dominator/proto/fleetmanager"
)

func (m *Manager) getLockedHypervisor(name string,
	writeLock bool) (*hypervisorType, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	if hypervisor, ok := m.hypervisors[name]; !ok {
		return nil, errors.New("Hypervisor not found")
	} else {
		if writeLock {
			hypervisor.mutex.Lock()
		} else {
			hypervisor.mutex.RLock()
		}
		return hypervisor, nil
	}
}

func (m *Manager) getHypervisorForVm(ipAddr net.IP) (string, error) {
	addr := ipAddr.String()
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	if vm, ok := m.vms[addr]; !ok {
		return "", errors.New("VM not found")
	} else {
		return vm.hypervisor.machine.Hostname, nil
	}
}

func (m *Manager) getMachineInfo(hostname string) (proto.Machine, error) {
	if !*manageHypervisors {
		return proto.Machine{},
			errors.New("this is a read-only Fleet Manager: full machine information is not available")
	}
	if hypervisor, err := m.getLockedHypervisor(hostname, false); err != nil {
		return proto.Machine{}, err
	} else {
		defer hypervisor.mutex.RUnlock()
		return *hypervisor.getMachineLocked(), nil
	}
}

func (m *Manager) getTopology() (*topology.Topology, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	if m.topology == nil {
		return nil, errors.New("no topology available")
	}
	return m.topology, nil
}
