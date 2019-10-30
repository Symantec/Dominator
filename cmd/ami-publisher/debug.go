package main

import (
	"runtime"

	"github.com/Cloud-Foundations/Dominator/lib/format"
	"github.com/Cloud-Foundations/Dominator/lib/log"
)

func logMemoryUsage(logger log.DebugLogger) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	logger.Debugf(0, "Memory: allocated: %s system: %s\n",
		format.FormatBytes(memStats.Alloc),
		format.FormatBytes(memStats.Sys))
}
