package manager

import (
	"errors"
	"os"
	"os/exec"
	"strconv"

	"github.com/Cloud-Foundations/Dominator/lib/meminfo"
)

var (
	errorInsufficientAvailableMemory = errors.New(
		"insufficient available memory")
	errorInsufficientUnallocatedMemory = errors.New(
		"insufficient unallocated memory")
	errorUnableToAllocatedMemory = errors.New("unable to allocate memory")
)

func checkAvailableMemory(memoryInMiB uint64) error {
	if memInfo, err := meminfo.GetMemInfo(); err != nil {
		return err
	} else {
		if memoryInMiB >= memInfo.Available>>20 {
			return errorInsufficientAvailableMemory
		}
		return nil
	}
}

func tryAllocateMemory(memoryInMiB uint64) <-chan error {
	channel := make(chan error, 1)
	executable, err := os.Executable()
	if err != nil {
		channel <- err
		return channel
	}
	cmd := exec.Command(executable, "-testMemoryAvailable",
		strconv.FormatUint(memoryInMiB, 10))
	go func() {
		if err := cmd.Run(); err != nil {
			if _, ok := err.(*exec.ExitError); ok {
				channel <- errorUnableToAllocatedMemory
			} else {
				channel <- err
			}
		} else {
			channel <- nil
		}
	}()
	return channel
}

func (m *Manager) getUnallocatedMemoryInMiB() uint64 {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.getUnallocatedMemoryInMiBWithLock()
}

func (m *Manager) getUnallocatedMemoryInMiBWithLock() uint64 {
	unallocated := int64(m.memTotalInMiB)
	for _, vm := range m.vms {
		unallocated -= int64(vm.MemoryInMiB)
	}
	if unallocated < 0 {
		return 0
	}
	return uint64(unallocated)
}

func (m *Manager) checkSufficientMemoryWithLock(memoryInMiB uint64) error {
	if memoryInMiB > m.getUnallocatedMemoryInMiBWithLock() {
		return errorInsufficientUnallocatedMemory
	}
	return checkAvailableMemory(memoryInMiB)
}
