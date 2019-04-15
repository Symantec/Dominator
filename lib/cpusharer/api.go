/*
	Package cpusharer implements co-operative CPU sharing between goroutines.

	Package cpusharer may be used by groups of co-operating goroutines to share
	CPU resources so that blocking operations are fully concurrent but avoiding
	the thundering herd problem when large numbers of goroutines need the CPU,
	impacting the responsiveness of other goroutines such as dashboards and
	health checks.
	Each goroutine calls the GrabCpu method when it starts and wraps blocking
	operations with a pair of ReleaseCpu/GrabCpu calls.
	A typical programming pattern is:
		cpuSharer := cpusharer.New*CpuSharer() // Pick your sharer of choice.
		for work := range workChannel {
			cpuSharer.GoWhenIdle(0, -1, func(work workType) {
				work.compute()
				cpuSharer.ReleaseCpu()
				work.block()
				cpuSharer.GrabCpu()
				work.moreCompute()
			}(work)
		}
*/
package cpusharer

import (
	"sync"
	"time"
)

// CpuSharer is the interface that wraps the GrabCpu and ReleaseCpu methods.
//
// GrabCpu will grab a CPU for use. If there are none available (i.e. all CPUs
// are in use by other co-operating goroutines) then this will block until a CPU
// is available.
//
// ReleaseCpu will release a CPU so that another co-operating goroutine can grab
// a CPU.
type CpuSharer interface {
	GrabCpu()
	ReleaseCpu()
}

type FifoCpuSharer struct {
	semaphore        chan struct{}
	mutex            sync.Mutex
	grabTimeout      time.Duration
	lastAcquireEvent time.Time
	lastIdleEvent    time.Time
	lastYieldEvent   time.Time
	numIdleEvents    uint64
	Statistics       Statistics
}

// NewFifoCpuSharer creates a simple FIFO CpuSharer. CPU access is granted in
// the order in which they are requested.
func NewFifoCpuSharer() *FifoCpuSharer {
	return newFifoCpuSharer()
}

// GetStatistics will update and return the Statistics.
func (s *FifoCpuSharer) GetStatistics() Statistics {
	return s.getStatistics()
}

// SetGrabTimeout will change the timeout for the GrabCpu method. A negative
// value for timeout means no timeout (this is the default). After the timeout a
// panic is generated.
// A full stack trace is written to os.Stderr.
func (s *FifoCpuSharer) SetGrabTimeout(timeout time.Duration) {
	s.setGrabTimeout(timeout)
}

// Go will start a goroutine and return. The goroutine will grab a CPU using the
// GrabCpu method, then will run goFunc. When goFunc returns the CPU is
// released.
func (s *FifoCpuSharer) Go(goFunc func()) {
	startGoroutine(s, goFunc)
}

// GoWhenAvailable will grab a CPU using the GrabCpu method and then starts a
// goroutine which will run goFunc. When goFunc returns the CPU is released.
// Use GoWhenAvailable to limit the addition of more goroutines if the CPUs are
// already saturated with work. This can reduce memory consumption spikes.
func (s *FifoCpuSharer) GoWhenAvailable(goFunc func()) {
	startGoroutineWhenAvailable(s, goFunc)
}

// GoWhenIdle is similar to the GoWhenAvailable method except that it will call
// GrabIdleCpu to wait for and grab an idle CPU. Compared to GoWhenAvailable,
// GoWhenIdle effectively lowers the priority of starting new goroutines below
// the priority of the co-operating goroutines. This can be even more effective
// in reducing memory consumption spikes.
// GoWhenIdle will wait up to timeout (if negative, it will wait forever) for an
// idle CPU. If an idle CPU is grabbed before the timeout it will return true,
// otherwise it will not start a goroutine and will return false.
func (s *FifoCpuSharer) GoWhenIdle(minIdleTime, timeout time.Duration,
	goFunc func()) bool {
	return s.goWhenIdle(minIdleTime, timeout, goFunc)
}

// GrabCpu will grab a CPU for use. If there are none available (i.e. all CPUs
// are in use by other co-operating goroutines) then this will block until a CPU
// is available. Grab requests are fulfilled in the order they are made.
func (s *FifoCpuSharer) GrabCpu() {
	s.grabCpu()
}

// GrabIdleCpu will wait for a CPU to be idle for at least minIdleTime and then
// grabs a CPU. If minIdleTime is zero or less then a CPU is grabbed immediately
// once one becomes idle. CPUs will never become idle while there are more
// goroutines blocked on GrabCpu than there are CPUs.
// GrabIdleCpu will wait up to timeout (if negative, it will wait forever) for
// an idle CPU. If an idle CPU is grabbed before the timeout it will return
// true, otherwise it will return false.
func (s *FifoCpuSharer) GrabIdleCpu(minIdleTime, timeout time.Duration) bool {
	return s.grabIdleCpu(minIdleTime, timeout)
}

// GrabSemaphore will safely grab the provided semaphore, releasing and
// re-acquiring the CPU if the semaphore blocks. Use this to avoid deadlocks.
func (s *FifoCpuSharer) GrabSemaphore(semaphore chan<- struct{}) {
	grabSemaphore(s, semaphore)
}

func (s *FifoCpuSharer) ReleaseCpu() {
	s.releaseCpu()
}

func (s *FifoCpuSharer) Sleep(duration time.Duration) {
	sleep(s, duration)
}

type Statistics struct {
	LastAcquireEvent time.Time
	LastIdleEvent    time.Time
	LastYieldEvent   time.Time
	NumCpuRunning    uint
	NumCpu           uint
	NumIdleEvents    uint64
}
