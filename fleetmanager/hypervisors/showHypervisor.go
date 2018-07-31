package hypervisors

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"sort"

	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/lib/url"
)

func (m *Manager) showHypervisorHandler(w http.ResponseWriter,
	req *http.Request) {
	parsedQuery := url.ParseQuery(req.URL)
	if len(parsedQuery.Flags) != 1 {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var hostname string
	for name := range parsedQuery.Flags {
		hostname = name
	}
	h, err := m.getLockedHypervisor(hostname, false)
	if err != nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		return
	}
	defer h.mutex.RUnlock()
	writer := bufio.NewWriter(w)
	defer writer.Flush()
	topology, err := m.getTopology()
	if err != nil {
		fmt.Fprintln(writer, err)
		return
	}
	fmt.Fprintf(writer, "<title>Information for Hypervisor %s</title>\n",
		hostname)
	fmt.Fprintln(writer, `<style>
                          table, th, td {
                          border-collapse: collapse;
                          }
                          </style>`)
	fmt.Fprintln(writer, "<body>")
	fmt.Fprintln(writer, "Machine info:<br>")
	fmt.Fprintln(writer, `<pre style="background-color: #eee; border: 1px solid #999; display: block; float: left;">`)
	json.WriteWithIndent(writer, "    ", h.machine)
	fmt.Fprintln(writer, `</pre><p style="clear: both;">`)
	subnets, err := topology.GetSubnetsForMachine(hostname)
	if err != nil {
		fmt.Fprintf(writer, "%s<br>\n", err)
	} else {
		fmt.Fprintln(writer, "Subnets:<br>")
		fmt.Fprintln(writer, `<pre style="background-color: #eee; border: 1px solid #999; display: block; float: left;">`)
		json.WriteWithIndent(writer, "    ", subnets)
		fmt.Fprintln(writer, `</pre><p style="clear: both;">`)
	}
	if !*manageHypervisors {
		fmt.Fprintln(writer, "No visibility into local tags<br>")
	} else if len(h.localTags) > 0 {
		keys := make([]string, 0, len(h.localTags))
		for key := range h.localTags {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		fmt.Fprintln(writer, "Local tags:<br>")
		fmt.Fprintln(writer, `<table border="1">`)
		fmt.Fprintln(writer, "  <tr>")
		fmt.Fprintln(writer, "    <th>Name</th>")
		fmt.Fprintln(writer, "    <th>Value</th>")
		fmt.Fprintln(writer, "  </tr>")
		for _, key := range keys {
			writeString(writer, key, h.localTags[key])
		}
		fmt.Fprintln(writer, "</table>")
	}
	fmt.Fprintf(writer,
		"Number of VMs known: <a href=\"http://%s:%d/listVMs\">%d</a>\n",
		hostname, constants.HypervisorPortNumber, len(h.vms))
	fmt.Fprintln(writer, "</body>")
}

func writeCountLinks(writer io.Writer, text, path string, count uint) {
	fmt.Fprintf(writer,
		"%s: <a href=\"%s\">%d</a> (<a href=\"%s&output=text\">text</a>)<br>\n",
		text, path, count, path)
}

func writeString(writer io.Writer, name, value string) {
	fmt.Fprintf(writer, "  <tr><td>%s</td><td>%s</td></tr>\n", name, value)
}
