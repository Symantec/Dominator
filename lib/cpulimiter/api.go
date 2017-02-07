package cpulimiter

import (
	"github.com/Symantec/Dominator/lib/wsyscall"
	"sync"
	"time"
)

type CpuLimiter struct {
	mutex            sync.Mutex
	confCpuPercent   uint // Aggregate across all CPUs.
	cpuPercent       uint // For a single CPU.
	lastProbeTime    time.Time
	lastProbeCpuTime wsyscall.Timeval
}

func New(cpuPercent uint) *CpuLimiter {
	return newCpuLimiter(cpuPercent)
}

func (cl *CpuLimiter) Limit() error {
	return cl.limit()
}

func (cl *CpuLimiter) CpuPercent() uint {
	return cl.getConfCpuPercent()
}

func (cl *CpuLimiter) SetCpuPercent(cpuPercent uint) {
	cl.setCpuPercent(cpuPercent)
}
