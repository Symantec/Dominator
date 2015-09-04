package herd

import (
	"fmt"
	"net"
	"net/http"
)

var httpdHerd *Herd

func (herd *Herd) startServer(portNum uint, daemon bool) error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", portNum))
	if err != nil {
		return err
	}
	httpdHerd = herd
	http.HandleFunc("/", statusHandler)
	http.HandleFunc("/listSubs", listSubsHandler)
	http.HandleFunc("/showSubs", showAllSubsHandler)
	http.HandleFunc("/showDeviantSubs", showDeviantSubsHandler)
	if daemon {
		go http.Serve(listener, nil)
	} else {
		http.Serve(listener, nil)
	}
	return nil
}

func (herd *Herd) addHtmlWriter(htmlWriter HtmlWriter) {
	herd.htmlWriters = append(herd.htmlWriters, htmlWriter)
}
