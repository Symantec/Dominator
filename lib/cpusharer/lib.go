package cpusharer

import (
	"time"
)

func grabSemaphore(s CpuSharer, semaphore chan<- struct{}) {
	select {
	case semaphore <- struct{}{}: // Non-blocking: don't release the CPU.
		return
	default:
		// Semaphore will block. Release the CPU and go back to the end of the
		// queue once the semaphore is grabbed.
		s.ReleaseCpu()
		semaphore <- struct{}{}
		s.GrabCpu()
	}
}

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
