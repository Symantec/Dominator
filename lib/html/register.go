package html

import (
	"bufio"
	"fmt"
	"net/http"
)

func registerHtmlWriterForPattern(pattern, title string,
	htmlWriter HtmlWriter) {
	http.HandleFunc(pattern,
		func(w http.ResponseWriter, req *http.Request) {
			writer := bufio.NewWriter(w)
			defer writer.Flush()
			fmt.Fprintf(writer, "<title>%s</title>\n", title)
			fmt.Fprintln(writer, "<body>")
			fmt.Fprintln(writer, "<center>")
			fmt.Fprintf(writer, "<h1>%s</h1>\n", title)
			fmt.Fprintln(writer, "</center>")
			htmlWriter.WriteHtml(writer)
			fmt.Fprintln(writer, "</body>")
		})
}
