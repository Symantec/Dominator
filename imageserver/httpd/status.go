package httpd

import (
	"bufio"
	"fmt"
	"net/http"
)

func statusHandler(w http.ResponseWriter, req *http.Request) {
	writer := bufio.NewWriter(w)
	defer writer.Flush()
	fmt.Fprintln(writer, "<title>imageserver status page</title>")
	fmt.Fprintln(writer, "<body>")
	fmt.Fprintln(writer, "<center>")
	fmt.Fprintln(writer, "<h1>imageserver status page</h1>")
	fmt.Fprintln(writer, "</center>")
	fmt.Fprintln(writer, "<h3>")
	writeLinks(writer)
	imageDataBase.WriteHtml(writer)
	fmt.Fprintln(writer, "</body>")
}
