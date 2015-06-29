package scanner

import (
	"bufio"
	"fmt"
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
	}
	fmt.Fprintf(w, "Duration of current scan: %s<br>\n",
		time.Since(fsh.timeOfLastScan))
	if fsh.generationCount > 0 {
		fmt.Fprintf(w, "Last change: %s<br>\n", fsh.timeOfLastChange)
	}
}
