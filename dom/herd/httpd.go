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
	http.HandleFunc("/listReachableSubs", listReachableSubsHandler)
	http.HandleFunc("/listSubs", listSubsHandler)
	http.HandleFunc("/showAliveSubs", showAliveSubsHandler)
	http.HandleFunc("/showAllSubs", showAllSubsHandler)
	http.HandleFunc("/showCompliantSubs", showCompliantSubsHandler)
	http.HandleFunc("/showDeviantSubs", showDeviantSubsHandler)
	http.HandleFunc("/showReachableSubs", showReachableSubsHandler)
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
