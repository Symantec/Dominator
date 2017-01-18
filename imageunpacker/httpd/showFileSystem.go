package httpd

import (
	"bufio"
	"fmt"
	"net/http"
)

func (s state) showFileSystemHandler(w http.ResponseWriter, req *http.Request) {
	writer := bufio.NewWriter(w)
	defer writer.Flush()
	streamName := req.URL.RawQuery
	fmt.Fprintf(writer, "<title>scanned stream  %s</title>\n", streamName)
	fmt.Fprintln(writer, "<body>")
	fmt.Fprintln(writer, "<h3>")
	fs, err := s.unpacker.GetFileSystem(streamName)
	if err != nil {
		fmt.Fprintln(writer, err)
	} else if fs == nil {
		fmt.Fprintln(writer, "No scanned file system")
	} else {
		fmt.Fprintf(writer, "Scanned file-system for stream: %s\n", streamName)
		fmt.Fprintln(writer, "</h3>")
		fmt.Fprintln(writer, "<pre>")
		fs.List(writer)
		fmt.Fprintln(writer, "</pre>")
	}
	fmt.Fprintln(writer, "</body>")
}
