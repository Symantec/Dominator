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
	http.HandleFunc("/", herd.statusHandler)
	http.HandleFunc("/listReachableSubs", herd.listReachableSubsHandler)
	http.HandleFunc("/listSubs", herd.listSubsHandler)
	http.HandleFunc("/showAliveSubs",
		html.BenchmarkedHandler(herd.showAliveSubsHandler))
	http.HandleFunc("/showAllSubs",
		html.BenchmarkedHandler(herd.showAllSubsHandler))
	http.HandleFunc("/showCompliantSubs",
		html.BenchmarkedHandler(herd.showCompliantSubsHandler))
	http.HandleFunc("/showDeviantSubs",
		html.BenchmarkedHandler(herd.showDeviantSubsHandler))
	http.HandleFunc("/showReachableSubs",
		html.BenchmarkedHandler(herd.showReachableSubsHandler))
	http.HandleFunc("/showSub", html.BenchmarkedHandler(herd.showSubHandler))
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
