package manager

import (
	"os"
	"sync"
	"time"

	proto "github.com/Symantec/Dominator/proto/hypervisor"
)

type flusher interface {
	Flush() error
}

func (m *Manager) shutdownVMsAndExit() {
	var waitGroup sync.WaitGroup
	m.mutex.RLock()
	for _, vm := range m.vms {
		waitGroup.Add(1)
		go func(vm *vmInfoType) {
			defer waitGroup.Done()
			vm.shutdown()
		}(vm)
	}
	waitGroup.Wait()
	m.Logger.Println("stopping cleanly after shutting down VMs")
	if flusher, ok := m.Logger.(flusher); ok {
		flusher.Flush()
	}
	os.Exit(0)
}

func (vm *vmInfoType) shutdown() {
	vm.mutex.RLock()
	switch vm.State {
	case proto.StateStarting, proto.StateRunning:
		stoppedNotifier := make(chan struct{}, 1)
		vm.stoppedNotifier = stoppedNotifier
		vm.commandChannel <- "system_powerdown"
		vm.mutex.RUnlock()
		timer := time.NewTimer(time.Minute)
		select {
		case <-stoppedNotifier:
			if !timer.Stop() {
				<-timer.C
			}
			vm.logger.Println("shut down cleanly for system shutdown")
		case <-timer.C:
			vm.logger.Println("shutdown timed out: killing VM")
			vm.commandChannel <- "quit"
		}
	default:
		vm.mutex.RUnlock()
	}
}
