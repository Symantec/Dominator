package memstats

import (
	"fmt"
	"io"
	"runtime"

	"github.com/Symantec/Dominator/lib/format"
)

func writeNamedStat(writer io.Writer, name string, value uint64) {
	fmt.Fprintf(writer, "  %s=%s\n", name, format.FormatBytes(value))
}

func WriteMemoryStats(writer io.Writer) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	fmt.Fprintln(writer, "MemStats:")
	writeNamedStat(writer, "Alloc", memStats.Alloc)
	writeNamedStat(writer, "Sys", memStats.Sys)
}
