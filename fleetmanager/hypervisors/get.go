package hypervisors

import (
	"errors"
	"net"

	"github.com/Symantec/Dominator/fleetmanager/topology"
)

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

func (m *Manager) getTopology() (*topology.Topology, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	if m.topology == nil {
		return nil, errors.New("no topology available")
	}
	return m.topology, nil
}
