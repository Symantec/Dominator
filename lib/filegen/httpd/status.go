package httpd

import (
	"bufio"
	"fmt"
	"net/http"

	"github.com/Symantec/Dominator/lib/html"
	"github.com/Symantec/Dominator/lib/srpc"
)

func (s *state) statusHandler(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path != "/" {
		http.NotFound(w, req)
		return
	}
	writer := bufio.NewWriter(w)
	defer writer.Flush()
	fmt.Fprintln(writer, "<title>Basic filegen server status page</title>")
	fmt.Fprintln(writer, "<body>")
	fmt.Fprintln(writer, "<center>")
	fmt.Fprintln(writer, "<h1>Basic filegen server status page</h1>")
	if !srpc.CheckTlsRequired() {
		fmt.Fprintln(writer,
			`<h1><font color="red">Running in insecure mode. Anyone can get your files!</font></h1>`)
	}
	fmt.Fprintln(writer, "</center>")
	html.WriteHeaderWithRequest(writer, req)
	fmt.Fprintln(writer, "<h3>")
	for _, htmlWriter := range htmlWriters {
		htmlWriter.WriteHtml(writer)
	}
	fmt.Fprintln(writer, "</h3>")
	fmt.Fprintln(writer, "<hr>")
	html.WriteFooter(writer)
	fmt.Fprintln(writer, "</body>")
}
