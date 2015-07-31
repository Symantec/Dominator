package httpd

import (
	"fmt"
	"github.com/Symantec/Dominator/sub/scanner"
	"io"
	"net"
	"net/http"
)

type HtmlWriter interface {
	WriteHtml(writer io.Writer)
}

var onlyHtmler HtmlWriter

func StartServer(portNum uint, fsh *scanner.FileSystemHistory) error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", portNum))
	if err != nil {
		return err
	}
	onlyHtmler = fsh
	http.HandleFunc("/", statusHandler)
	go http.Serve(listener, nil)
	return nil
}
