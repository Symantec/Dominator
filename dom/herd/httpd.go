package herd

import (
	"fmt"
	"net"
	"net/http"
)

func (herd *Herd) startServer(portNum uint, daemon bool) error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", portNum))
	if err != nil {
		return err
	}
	http.HandleFunc("/", herd.statusHandler)
	http.HandleFunc("/listReachableSubs", herd.listReachableSubsHandler)
	http.HandleFunc("/listSubs", herd.listSubsHandler)
	http.HandleFunc("/showAliveSubs", herd.showAliveSubsHandler)
	http.HandleFunc("/showAllSubs", herd.showAllSubsHandler)
	http.HandleFunc("/showCompliantSubs", herd.showCompliantSubsHandler)
	http.HandleFunc("/showDeviantSubs", herd.showDeviantSubsHandler)
	http.HandleFunc("/showReachableSubs", herd.showReachableSubsHandler)
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
