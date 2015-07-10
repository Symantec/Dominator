package memstats

import (
	"fmt"
	"github.com/Symantec/Dominator/sub/fsrateio"
	"io"
	"runtime"
)

func writeNamedStat(writer io.Writer, name string, value uint64) {
	fmt.Fprintf(writer, "  %s=%s\n", name, fsrateio.FormatBytes(value))
}

func WriteMemoryStats(writer io.Writer) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	fmt.Fprintln(writer, "MemStats:")
	writeNamedStat(writer, "Alloc", memStats.Alloc)
	writeNamedStat(writer, "TotalAlloc", memStats.TotalAlloc)
	writeNamedStat(writer, "Sys", memStats.Sys)
}
