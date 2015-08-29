package herd

import (
	"bufio"
	"fmt"
	"github.com/Symantec/Dominator/lib/html"
	"io"
	"net/http"
	"time"
)

func statusHandler(w http.ResponseWriter, req *http.Request) {
	writer := bufio.NewWriter(w)
	defer writer.Flush()
	fmt.Fprintln(writer, "<title>Dominator status page</title>")
	fmt.Fprintln(writer, "<body>")
	fmt.Fprintln(writer, "<center>")
	fmt.Fprintln(writer, "<h1><b>Dominator</b> status page</h1>")
	fmt.Fprintln(writer, "</center>")
	html.WriteHeader(writer)
	fmt.Fprintln(writer, "<h3>")
	writeStatus(writer, httpdHerd)
	fmt.Fprintln(writer, "</h3>")
	fmt.Fprintln(writer, "<hr>")
	html.WriteFooter(writer)
	fmt.Fprintln(writer, "</body>")
}

func writeStatus(writer io.Writer, herd *Herd) {
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
