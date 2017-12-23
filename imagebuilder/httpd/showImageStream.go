package httpd

import (
	"bufio"
	"fmt"
	"net/http"
)

func (s state) showImageStreamHandler(w http.ResponseWriter,
	req *http.Request) {
	writer := bufio.NewWriter(w)
	defer writer.Flush()
	streamName := req.URL.RawQuery
	fmt.Fprintf(writer, "<title>image stream %s</title>\n", streamName)
	fmt.Fprintln(writer, "<body>")
	s.builder.ShowImageStream(writer, streamName)
	fmt.Fprintln(writer, "</body>")
}
