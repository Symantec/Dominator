package cpulimiter

import (
	"github.com/Symantec/Dominator/lib/wsyscall"
	"runtime"
	"time"
)

var minCheckInterval = time.Millisecond * 10

func newCpuLimiter(cpuPercent uint) *CpuLimiter {
	cl := new(CpuLimiter)
	cl.setCpuPercent(cpuPercent)
	return cl
}

func (cl *CpuLimiter) limit() error {
	if cl.cpuPercent >= 100 {
		return nil
	}
	now := time.Now()
	if cl.lastProbeTime.IsZero() { // Initialise.
		var rusage wsyscall.Rusage
		err := wsyscall.Getrusage(wsyscall.RUSAGE_THREAD, &rusage)
		if err != nil {
			return err
		}
		cl.lastProbeTime = now
		cl.lastProbeCpuTime = rusage.Utime
		return nil
	}
	wallTimeSinceLastProbe := now.Sub(cl.lastProbeTime)
	if wallTimeSinceLastProbe < minCheckInterval {
		return nil
	}
	var rusage wsyscall.Rusage
	if err := wsyscall.Getrusage(wsyscall.RUSAGE_THREAD, &rusage); err != nil {
		return err
	}
	cpuTimeSinceLastProbe := time.Duration(rusage.Utime.Sec-
		cl.lastProbeCpuTime.Sec) * time.Second
	cpuTimeSinceLastProbe += time.Duration(
		rusage.Utime.Usec-cl.lastProbeCpuTime.Usec) * time.Microsecond
	time.Sleep(cpuTimeSinceLastProbe*100/time.Duration(cl.cpuPercent) -
		wallTimeSinceLastProbe)
	cl.lastProbeTime = time.Now()
	cl.lastProbeCpuTime = rusage.Utime
	return nil
}

func (cl *CpuLimiter) setCpuPercent(cpuPercent uint) {
	if cpuPercent < 1 {
		cpuPercent = 1
	}
	cpuPercent *= uint(runtime.NumCPU())
	if cpuPercent > 100 {
		cpuPercent = 100
	}
	cl.cpuPercent = cpuPercent
}
