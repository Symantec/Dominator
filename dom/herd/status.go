package herd

import (
	"bufio"
	"fmt"
	"github.com/Symantec/Dominator/lib/html"
	"github.com/Symantec/Dominator/lib/srpc"
	"net/http"
)

func (herd *Herd) statusHandler(w http.ResponseWriter, req *http.Request) {
	writer := bufio.NewWriter(w)
	defer writer.Flush()
	fmt.Fprintln(writer, "<title>Dominator status page</title>")
	fmt.Fprintln(writer, "<body>")
	fmt.Fprintln(writer, "<center>")
	fmt.Fprintln(writer, "<h1><b>Dominator</b> status page</h1>")
	if !srpc.CheckTlsRequired() {
		fmt.Fprintln(writer,
			`<h1><font color="red">Running in insecure mode. You can get pwned!!!</font></h1>`)
	}
	fmt.Fprintln(writer, "</center>")
	html.WriteHeaderWithRequestNoGC(writer, req)
	fmt.Fprintln(writer, "<h3>")
	herd.writeHtml(writer)
	for _, htmlWriter := range herd.htmlWriters {
		htmlWriter.WriteHtml(writer)
	}
	fmt.Fprintln(writer, "</h3>")
	fmt.Fprintln(writer, "<hr>")
	html.WriteFooter(writer)
	fmt.Fprintln(writer, "</body>")
}
