package herd

import (
	"fmt"
	"io"
	"time"
)

func (herd *Herd) writeHtml(writer io.Writer) {
	fmt.Fprintf(writer, "Duration of current scan cycle: %s<br>\n",
		time.Since(herd.currentScanStartTime))
	fmt.Fprintf(writer, "Duration of previous scan cycle: %s<br>\n",
		herd.currentScanStartTime.Sub(herd.previousScanStartTime))
	fmt.Fprintf(writer, "Image server: <a href=\"http://%s/\">%s</a><br>\n",
		herd.imageServerAddress, herd.imageServerAddress)
	herd.RLock()
	numSubs := len(herd.subsByName)
	herd.RUnlock()
	fmt.Fprintf(writer,
		"Number of <a href=\"listSubs\">subs</a>: <a href=\"showSubs\">%d</a><br>\n",
		numSubs)
	fmt.Fprintf(writer, "Connection slots: %d out of %d<br>\n",
		len(herd.makeConnectionSemaphore), cap(herd.makeConnectionSemaphore))
	fmt.Fprintf(writer, "RPC slots: %d out of %d<br>\n",
		len(herd.pollSemaphore), cap(herd.pollSemaphore))
}
