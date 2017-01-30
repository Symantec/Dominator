package cpulimiter

import (
	"github.com/Symantec/Dominator/lib/wsyscall"
	"time"
)

type CpuLimiter struct {
	cpuPercent       uint
	lastProbeTime    time.Time
	lastProbeCpuTime wsyscall.Timeval
}

func New(cpuPercent uint) *CpuLimiter {
	return newCpuLimiter(cpuPercent)
}

func (cl *CpuLimiter) Limit() error {
	return cl.limit()
}

func (cl *CpuLimiter) CpuPercent() uint { return cl.cpuPercent }

func (cl *CpuLimiter) SetCpuPercent(cpuPercent uint) {
	cl.setCpuPercent(cpuPercent)
}
