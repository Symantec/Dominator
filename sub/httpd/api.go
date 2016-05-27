package httpd

import (
	"fmt"
	"io"
	"net"
	"net/http"
)

type HtmlWriter interface {
	WriteHtml(writer io.Writer)
}

var htmlWriters []HtmlWriter
var insecureMode bool

func StartServer(portNum uint, isInsecure bool) error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", portNum))
	if err != nil {
		return err
	}
	insecureMode = isInsecure
	http.HandleFunc("/", statusHandler)
	go http.Serve(listener, nil)
	return nil
}

func AddHtmlWriter(htmlWriter HtmlWriter) {
	htmlWriters = append(htmlWriters, htmlWriter)
}
