package cpusharer

func (s *FifoCpuSharer) getStatistics() Statistics {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.Statistics.NumCpuRunning = uint(len(s.semaphore))
	s.Statistics.NumCpu = uint(cap(s.semaphore))
	s.Statistics.NumIdleEvents = s.numIdleEvents
	return s.Statistics
}

func (s *FifoCpuSharer) grabCpu() {
	select {
	case s.semaphore <- struct{}{}:
		s.mutex.Lock()
		defer s.mutex.Unlock()
		s.numIdleEvents++
		return
	default:
		s.semaphore <- struct{}{}
	}
}
