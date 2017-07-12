package cpusharer

import (
	"os"
	"runtime"
	"time"
)

var timeoutMessage = []byte("CPU grabber timeout. Full stack trace follows:\n")

func newFifoCpuSharer() *FifoCpuSharer {
	return &FifoCpuSharer{
		semaphore:     make(chan struct{}, runtime.NumCPU()),
		grabTimeout:   -1,
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
	default: // No CPU is available yet: block waiting with timeout.
		s.mutex.Lock()
		timeout := s.grabTimeout
		s.mutex.Unlock()
		if timeout < 0 {
			s.semaphore <- struct{}{} // Block forever waiting for a CPU.
			return
		}
		timer := time.NewTimer(timeout)
		select {
		case s.semaphore <- struct{}{}: // A CPU became available.
			if !timer.Stop() {
				<-timer.C
			}
			return
		case <-timer.C:
			os.Stderr.Write(timeoutMessage)
			buf := make([]byte, 1024*1024)
			nBytes := runtime.Stack(buf, true)
			os.Stderr.Write(buf[0:nBytes])
			os.Stderr.Write([]byte("\n"))
			panic("timeout")
		}
	}
}

func (s *FifoCpuSharer) setGrabTimeout(timeout time.Duration) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.grabTimeout = timeout
}
