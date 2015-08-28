package herd

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"sort"
)

func listSubsHandler(w http.ResponseWriter, req *http.Request) {
	writer := bufio.NewWriter(w)
	defer writer.Flush()
	listSubs(writer)
}

func listSubs(writer io.Writer) {
	httpdHerd.RLock()
	subs := make([]string, 0, len(httpdHerd.subsByName))
	for name, _ := range httpdHerd.subsByName {
		subs = append(subs, name)
	}
	httpdHerd.RUnlock()
	sort.Strings(subs)
	for _, name := range subs {
		fmt.Fprintln(writer, name)
	}
}
