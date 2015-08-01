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

var onlyHtmler HtmlWriter

func StartServer(portNum uint, htmlWriter HtmlWriter) error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", portNum))
	if err != nil {
		return err
	}
	onlyHtmler = htmlWriter
	http.HandleFunc("/", statusHandler)
	go http.Serve(listener, nil)
	return nil
}
