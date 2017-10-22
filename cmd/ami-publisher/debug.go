package main

import (
	"runtime"

	"github.com/Symantec/Dominator/lib/format"
	"github.com/Symantec/Dominator/lib/log"
)

func logMemoryUsage(logger log.DebugLogger) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	logger.Debugf(0, "Memory: allocated: %s system: %s\n",
		format.FormatBytes(memStats.Alloc),
		format.FormatBytes(memStats.Sys))
}
