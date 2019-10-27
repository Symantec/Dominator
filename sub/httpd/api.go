package httpd

import (
	"io"
	"net/http"

	"github.com/Cloud-Foundations/Dominator/lib/html"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/net/reverseconnection"
	"github.com/Cloud-Foundations/tricorder/go/tricorder"
)

type HtmlWriter interface {
	WriteHtml(writer io.Writer)
}

var htmlWriters []HtmlWriter

func StartServer(portNum uint, logger log.DebugLogger) error {
	listener, err := reverseconnection.Listen("tcp", portNum, logger)
	if err != nil {
		return err
	}
	if err := listener.RequestConnections("Dominator"); err != nil {
		return err
	}
	err = listener.RequestConnections(tricorder.CollectorServiceName)
	if err != nil {
		return err
	}
	html.HandleFunc("/", statusHandler)
	go http.Serve(listener, nil)
	return nil
}

func AddHtmlWriter(htmlWriter HtmlWriter) {
	htmlWriters = append(htmlWriters, htmlWriter)
}
