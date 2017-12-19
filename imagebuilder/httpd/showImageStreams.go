package httpd

import (
	"bufio"
	"fmt"
	"net/http"
)

func (s state) showImageStreamsHandler(w http.ResponseWriter,
	req *http.Request) {
	writer := bufio.NewWriter(w)
	defer writer.Flush()
	fmt.Fprintln(writer, "<title>imaginator image streams</title>")
	fmt.Fprintln(writer, `<style>
                          table, th, td {
                          border-collapse: collapse;
                          }
                          </style>`)
	fmt.Fprintln(writer, "<body>")
	fmt.Fprintln(writer, "<h3>")
	s.builder.ShowImageStreams(writer)
	fmt.Fprintln(writer, "</body>")
}
