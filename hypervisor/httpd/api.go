package httpd

import (
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/Cloud-Foundations/Dominator/hypervisor/manager"
	"github.com/Cloud-Foundations/Dominator/lib/html"
)

type HtmlWriter interface {
	WriteHtml(writer io.Writer)
}

var htmlWriters []HtmlWriter

type state struct {
	manager *manager.Manager
}

func StartServer(portNum uint, managerObj *manager.Manager, daemon bool) error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", portNum))
	if err != nil {
		return err
	}
	myState := state{managerObj}
	html.HandleFunc("/", myState.statusHandler)
	html.HandleFunc("/listAvailableAddresses",
		myState.listAvailableAddressesHandler)
	html.HandleFunc("/listSubnets", myState.listSubnetsHandler)
	html.HandleFunc("/listVMs", myState.listVMsHandler)
	html.HandleFunc("/showVmBootLog", myState.showBootLogHandler)
	html.HandleFunc("/showVM", myState.showVMHandler)
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
