package logutil

import (
	"runtime"

	"github.com/Symantec/Dominator/lib/format"
	"github.com/Symantec/Dominator/lib/log"
)

func logMemory(logger log.DebugLogger, level int16, message string) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	allocMem := format.FormatBytes(memStats.Alloc)
	sysMem := format.FormatBytes(memStats.Sys - memStats.HeapReleased)
	if level < 0 {
		logger.Printf("%s: memory: allocated: %s, system: %s\n",
			message, allocMem, sysMem)
	} else {
		logger.Debugf(uint8(level), "%s: memory: allocated: %s, system: %s\n",
			message, allocMem, sysMem)
	}
}
