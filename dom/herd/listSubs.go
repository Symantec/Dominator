package herd

import (
	"bufio"
	"fmt"
	"github.com/Symantec/Dominator/lib/url"
	"net/http"
)

func (herd *Herd) listReachableSubsHandler(w http.ResponseWriter,
	req *http.Request) {
	writer := bufio.NewWriter(w)
	defer writer.Flush()
	selector, err := herd.getReachableSelector(url.ParseQuery(req.URL))
	if err != nil {
		fmt.Fprintln(writer, err)
		return
	}
	for _, sub := range herd.getSelectedSubs(selector) {
		fmt.Fprintln(writer, sub.mdb.Hostname)
	}
}

func (herd *Herd) listSubsHandler(w http.ResponseWriter, req *http.Request) {
	writer := bufio.NewWriter(w)
	defer writer.Flush()
	herd.RLock()
	subs := make([]string, 0, len(herd.subsByIndex))
	for _, sub := range herd.subsByIndex {
		subs = append(subs, sub.mdb.Hostname)
	}
	herd.RUnlock()
	for _, name := range subs {
		fmt.Fprintln(writer, name)
	}
}
