package manager

import (
	"errors"
)

var (
	errorInsufficientUnallocatedCPU = errors.New(
		"insufficient unallocated CPU")
)

func (m *Manager) checkSufficientCPUWithLock(milliCPU uint) error {
	if milliCPU > m.getAvailableMilliCPUWithLock() {
		return errorInsufficientUnallocatedCPU
	}
	return nil
}

func (m *Manager) getAvailableMilliCPU() uint {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.getAvailableMilliCPUWithLock()
}

func (m *Manager) getAvailableMilliCPUWithLock() uint {
	available := m.numCPU * 1000
	for _, vm := range m.vms {
		available -= int(vm.MilliCPUs)
	}
	if available < 0 {
		return 0
	}
	return uint(available)
}
