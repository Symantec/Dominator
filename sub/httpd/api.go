package httpd

import (
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/Symantec/Dominator/lib/log"
)

type HtmlWriter interface {
	WriteHtml(writer io.Writer)
}

var htmlWriters []HtmlWriter

func StartServer(portNum uint, logger log.DebugLogger) error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", portNum))
	if err != nil {
		return err
	}
	http.HandleFunc("/", statusHandler)
	go http.Serve(listener, nil)
	return nil
}

func AddHtmlWriter(htmlWriter HtmlWriter) {
	htmlWriters = append(htmlWriters, htmlWriter)
}
