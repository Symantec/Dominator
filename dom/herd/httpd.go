package herd

import (
	"fmt"
	"net"
	"net/http"

	"github.com/Symantec/Dominator/lib/html"
)

func (herd *Herd) startServer(portNum uint, daemon bool) error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", portNum))
	if err != nil {
		return err
	}
	html.HandleFunc("/", herd.statusHandler)
	html.HandleFunc("/listReachableSubs", herd.listReachableSubsHandler)
	html.HandleFunc("/listSubs", herd.listSubsHandler)
	html.HandleFunc("/showAliveSubs",
		html.BenchmarkedHandler(herd.showAliveSubsHandler))
	html.HandleFunc("/showAllSubs",
		html.BenchmarkedHandler(herd.showAllSubsHandler))
	html.HandleFunc("/showCompliantSubs",
		html.BenchmarkedHandler(herd.showCompliantSubsHandler))
	html.HandleFunc("/showDeviantSubs",
		html.BenchmarkedHandler(herd.showDeviantSubsHandler))
	html.HandleFunc("/showReachableSubs",
		html.BenchmarkedHandler(herd.showReachableSubsHandler))
	html.HandleFunc("/showSub", html.BenchmarkedHandler(herd.showSubHandler))
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
