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

func StartServer(portNum uint, htmler HtmlWriter) error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", portNum))
	if err != nil {
		return err
	}
	onlyHtmler = htmler
	http.HandleFunc("/", onlyHandler)
	go http.Serve(listener, nil)
	return nil
}

func onlyHandler(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintln(w, "<title>subd status page</title>")
	fmt.Fprintln(w, "<body>")
	fmt.Fprintln(w, "<center>")
	fmt.Fprintln(w, "<h1>subd status page</h1>")
	fmt.Fprintln(w, "</center>")
	fmt.Fprintln(w, "<h3>")
	onlyHtmler.WriteHtml(w)
	fmt.Fprintln(w, "</body>")
}
