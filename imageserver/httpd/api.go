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

func StartServer(portNum uint, htmlWriter HtmlWriter, daemon bool) error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", portNum))
	if err != nil {
		return err
	}
	onlyHtmler = htmlWriter
	http.HandleFunc("/", statusHandler)
	if daemon {
		go http.Serve(listener, nil)
	} else {
		http.Serve(listener, nil)
	}
	return nil
}
