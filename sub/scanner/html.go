package scanner

import (
	"bufio"
	"fmt"
	"github.com/Symantec/Dominator/sub/fsrateio"
	"io"
	"time"
)

func (fsh *FileSystemHistory) writeHtml(writer io.Writer) {
	w := bufio.NewWriter(writer)
	defer w.Flush()
	fmt.Fprintf(w, "Scan count: %d<br>\n", fsh.scanCount)
	fmt.Fprintf(w, "Generation count: %d<br>\n", fsh.generationCount)
	if fsh.scanCount > 0 {
		fmt.Fprintf(w, "Last scan completed: %s<br>\n", fsh.timeOfLastScan)
		fmt.Fprintf(w, "Duration of last scan: %s<br>\n",
			fsh.durationOfLastScan)
		tmp := fsrateio.FormatBytes(fsh.fileSystem.TotalDataBytes)
		fmt.Fprintf(w, "Scanned: %s<br>\n", tmp)
		tmp = fsrateio.FormatBytes(uint64(float64(
			fsh.fileSystem.TotalDataBytes) / fsh.durationOfLastScan.Seconds()))
		fmt.Fprintf(w, "Scan rate: %s/s<br>\n", tmp)
	}
	fmt.Fprintf(w, "Duration of current scan: %s<br>\n",
		time.Since(fsh.timeOfLastScan))
	if fsh.generationCount > 0 {
		fmt.Fprintf(w, "Last change: %s<br>\n", fsh.timeOfLastChange)
	}
}
