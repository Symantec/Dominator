package scanner

import (
	"github.com/Symantec/Dominator/lib/log"
)

func (configuration *Configuration) boostCpuLimit(logger log.Logger) {
	if configuration.CpuLimiter != nil {
		cl := configuration.CpuLimiter
		if cl.CpuPercent() != 100 {
			logger.Println("Boosting CPU limit: 100%")
		}
		cl.SetCpuPercent(100)
	}
}

func (configuration *Configuration) restoreCpuLimit(logger log.Logger) {
	if configuration.CpuLimiter != nil && configuration.DefaultCpuPercent > 0 {
		cl := configuration.CpuLimiter
		if cl.CpuPercent() != configuration.DefaultCpuPercent {
			logger.Printf("Restoring CPU limit: %d%%\n",
				configuration.DefaultCpuPercent)
		}
		cl.SetCpuPercent(configuration.DefaultCpuPercent)
	}
}
