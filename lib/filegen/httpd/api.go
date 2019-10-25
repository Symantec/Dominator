package httpd

import (
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/Cloud-Foundations/Dominator/lib/filegen"
	"github.com/Cloud-Foundations/Dominator/lib/html"
)

type HtmlWriter interface {
	WriteHtml(writer io.Writer)
}

var htmlWriters []HtmlWriter

type state struct {
	manager *filegen.Manager
}

func StartServer(portNum uint, manager *filegen.Manager, daemon bool) error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", portNum))
	if err != nil {
		return err
	}
	myState := &state{manager}
	html.HandleFunc("/", myState.statusHandler)
	html.HandleFunc("/listGenerators", myState.listGeneratorsHandler)
	if daemon {
		go http.Serve(listener, nil)
	} else {
		http.Serve(listener, nil)
	}
	return nil
}

func AddHtmlWriter(htmlWriter HtmlWriter) {
	htmlWriters = append(htmlWriters, htmlWriter)
}
