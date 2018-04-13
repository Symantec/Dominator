package httpd

import (
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/Symantec/Dominator/hypervisor/manager"
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
	http.HandleFunc("/", myState.statusHandler)
	http.HandleFunc("/listAvailableAddresses",
		myState.listAvailableAddressesHandler)
	http.HandleFunc("/listSubnets", myState.listSubnetsHandler)
	http.HandleFunc("/listVMs", myState.listVMsHandler)
	http.HandleFunc("/showVmBootLog", myState.showBootLogHandler)
	http.HandleFunc("/showVM", myState.showVMHandler)
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
