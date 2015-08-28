package herd

import (
	"bufio"
	"fmt"
	"github.com/Symantec/Dominator/lib/html"
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
	fmt.Fprintf(writer, "Duration of current scan cycle: %s<br>\n",
		time.Since(httpdHerd.currentScanStartTime))
	fmt.Fprintf(writer, "Duration of previous scan cycle: %s<br>\n",
		httpdHerd.currentScanStartTime.Sub(httpdHerd.previousScanStartTime))
	fmt.Fprintln(writer, "</h3>")
	fmt.Fprintln(writer, "<hr>")
	html.WriteFooter(writer)
	fmt.Fprintln(writer, "</body>")
}
