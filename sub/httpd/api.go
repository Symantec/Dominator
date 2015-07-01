package httpd

import (
	"fmt"
	"github.com/Symantec/Dominator/sub/scanner"
	"io"
	"net"
	"net/http"
	"net/rpc"
)

type HtmlWriter interface {
	WriteHtml(writer io.Writer)
}

var onlyHtmler HtmlWriter
var onlyFsh *scanner.FileSystemHistory

type Subd int

func (t *Subd) Poll(generation uint64, reply *scanner.FileSystem) error {
	if onlyFsh.FileSystem() != nil {
		*reply = *onlyFsh.FileSystem()
	}
	return nil
}

func StartServer(portNum uint, fsh *scanner.FileSystemHistory) error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", portNum))
	if err != nil {
		return err
	}
	onlyHtmler = fsh
	onlyFsh = fsh
	http.HandleFunc("/", onlyHandler)
	subd := new(Subd)
	rpc.Register(subd)
	rpc.HandleHTTP()
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
