package httpd

import (
	"bufio"
	"fmt"
	"net/http"
)

func statusHandler(w http.ResponseWriter, req *http.Request) {
	writer := bufio.NewWriter(w)
	defer writer.Flush()
	fmt.Fprintln(writer, "<title>subd status page</title>")
	fmt.Fprintln(writer, "<body>")
	fmt.Fprintln(writer, "<center>")
	fmt.Fprintln(writer, "<h1>subd status page</h1>")
	fmt.Fprintln(writer, "</center>")
	fmt.Fprintln(writer, "<h3>")
	onlyHtmler.WriteHtml(writer)
	fmt.Fprintln(writer, "</body>")
}
