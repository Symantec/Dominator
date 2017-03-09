package cpusharer

import (
	"runtime"
	"time"
)

func newFifoCpuSharer() *FifoCpuSharer {
	return &FifoCpuSharer{
		semaphore:     make(chan struct{}, runtime.NumCPU()),
		lastIdleEvent: time.Now(),
	}
}

func (s *FifoCpuSharer) getStatistics() Statistics {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.Statistics.LastIdleEvent = s.lastIdleEvent
	s.Statistics.NumCpuRunning = uint(len(s.semaphore))
	s.Statistics.NumCpu = uint(cap(s.semaphore))
	s.Statistics.NumIdleEvents = s.numIdleEvents
	return s.Statistics
}

func (s *FifoCpuSharer) grabCpu() {
	select {
	case s.semaphore <- struct{}{}: // A CPU is immediately available.
		s.mutex.Lock()
		defer s.mutex.Unlock()
		s.lastIdleEvent = time.Now()
		s.numIdleEvents++
		return
	default: // No CPU is available yet: block waiting.
		s.semaphore <- struct{}{}
	}
}
