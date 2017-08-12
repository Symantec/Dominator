package httpd

import (
	"fmt"
	"github.com/Symantec/Dominator/imagebuilder/builder"
	"io"
	"net"
	"net/http"
)

type HtmlWriter interface {
	WriteHtml(writer io.Writer)
}

var htmlWriters []HtmlWriter

type state struct {
	builder *builder.Builder
}

func StartServer(portNum uint, builderObj *builder.Builder,
	daemon bool) error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", portNum))
	if err != nil {
		return err
	}
	myState := state{builderObj}
	http.HandleFunc("/", myState.statusHandler)
	http.HandleFunc("/showCurrentBuildLog", myState.showCurrentBuildLogHandler)
	http.HandleFunc("/showLastBuildLog", myState.showLastBuildLogHandler)
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
