package cpusharer

import (
	"fmt"
	"github.com/Symantec/Dominator/lib/format"
	"os"
	"runtime"
	"time"
)

func newFifoCpuSharer() *FifoCpuSharer {
	return &FifoCpuSharer{
		semaphore:        make(chan struct{}, runtime.NumCPU()),
		grabTimeout:      -1,
		lastAcquireEvent: time.Now(),
		lastIdleEvent:    time.Now(),
		lastYieldEvent:   time.Now(),
	}
}

func (s *FifoCpuSharer) getStatistics() Statistics {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.Statistics.LastAcquireEvent = s.lastAcquireEvent
	s.Statistics.LastIdleEvent = s.lastIdleEvent
	s.Statistics.LastYieldEvent = s.lastYieldEvent
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
		s.lastAcquireEvent = time.Now()
		s.lastIdleEvent = s.lastAcquireEvent
		s.numIdleEvents++
		return
	default: // No CPU is available yet: block waiting with timeout.
		s.mutex.Lock()
		timeout := s.grabTimeout
		s.mutex.Unlock()
		if timeout < 0 {
			s.semaphore <- struct{}{} // Block forever waiting for a CPU.
			s.mutex.Lock()
			defer s.mutex.Unlock()
			s.lastAcquireEvent = time.Now()
			return
		}
		timer := time.NewTimer(timeout)
		select {
		case s.semaphore <- struct{}{}: // A CPU became available.
			if !timer.Stop() {
				<-timer.C
			}
			s.mutex.Lock()
			defer s.mutex.Unlock()
			s.lastAcquireEvent = time.Now()
			return
		case <-timer.C:
			stats := s.GetStatistics()
			fmt.Fprintf(os.Stderr,
				"CPU grabber timeout. Last acquire: %s, last yield: %s\n",
				format.Duration(time.Since(stats.LastAcquireEvent)),
				format.Duration(time.Since(stats.LastYieldEvent)))
			fmt.Fprintln(os.Stderr, "Full stack track follows:")
			buf := make([]byte, 1024*1024)
			nBytes := runtime.Stack(buf, true)
			os.Stderr.Write(buf[0:nBytes])
			os.Stderr.Write([]byte("\n"))
			panic("timeout")
		}
	}
}

func (s *FifoCpuSharer) releaseCpu() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.lastYieldEvent = time.Now()
	<-s.semaphore
}

func (s *FifoCpuSharer) setGrabTimeout(timeout time.Duration) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.grabTimeout = timeout
}
