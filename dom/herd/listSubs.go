package herd

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
)

func listSubsHandler(w http.ResponseWriter, req *http.Request) {
	writer := bufio.NewWriter(w)
	defer writer.Flush()
	listSubs(writer)
}

func listSubs(writer io.Writer) {
	httpdHerd.RLock()
	subs := make([]string, 0, len(httpdHerd.subsByIndex))
	for _, sub := range httpdHerd.subsByIndex {
		subs = append(subs, sub.hostname)
	}
	httpdHerd.RUnlock()
	for _, name := range subs {
		fmt.Fprintln(writer, name)
	}
}
