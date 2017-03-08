package cpusharer

import (
	"time"
)

func startGoroutine(s CpuSharer, goFunc func()) {
	go func() {
		s.GrabCpu()
		goFunc()
		s.ReleaseCpu()
	}()
}

func sleep(s CpuSharer, duration time.Duration) {
	s.ReleaseCpu()
	time.Sleep(duration)
	s.GrabCpu()
}
