package herd

import (
	"bufio"
	"fmt"
	"github.com/Symantec/Dominator/lib/constants"
	"io"
	"net/http"
	"strings"
	"time"
)

func showSubsHandler(w http.ResponseWriter, req *http.Request) {
	writer := bufio.NewWriter(w)
	defer writer.Flush()
	fmt.Fprintln(writer, "<title>Dominator subs</title>")
	fmt.Fprintln(writer, `<style>
                          table, th, td {
                          border-collapse: collapse;
                          }
                          </style>`)
	fmt.Fprintln(writer, "<body>")
	fmt.Fprintln(writer, "<h3>")
	fmt.Fprintln(writer, `<table border="1" style="width:100%">`)
	fmt.Fprintln(writer, "  <tr>")
	fmt.Fprintln(writer, "    <th>Name</b></th>")
	fmt.Fprintln(writer, "    <th>Required Image</b></th>")
	fmt.Fprintln(writer, "    <th>Planned Image</b></th>")
	fmt.Fprintln(writer, "    <th>Busy</b></th>")
	fmt.Fprintln(writer, "    <th>Status</b></th>")
	fmt.Fprintln(writer, "    <th>Staleness</b></th>")
	fmt.Fprintln(writer, "  </tr>")
	subs := httpdHerd.getSortedSubs(nil)
	for _, sub := range subs {
		showSub(writer, sub)
	}
	fmt.Fprintln(writer, "</table>")
	fmt.Fprintln(writer, "</body>")
}

func showSub(writer io.Writer, sub *Sub) {
	fmt.Fprintf(writer, "  <tr>\n")
	subURL := fmt.Sprintf("http://%s:%d/",
		strings.SplitN(sub.hostname, "*", 2)[0], constants.SubPortNumber)
	fmt.Fprintf(writer, "    <td><a href=\"%s\">%s</a></td>\n",
		subURL, sub.hostname)
	fmt.Fprintf(writer, "    <td>%s</td>\n", sub.requiredImage)
	fmt.Fprintf(writer, "    <td>%s</td>\n", sub.plannedImage)
	fmt.Fprintf(writer, "    <td>%v</td>\n", sub.busy)
	var status string
	switch {
	case sub.status == statusUnknown:
		status = "unknown"
	case sub.status == statusConnecting:
		status = "connecting"
	case sub.status == statusFailedToConnect:
		status = "connect failed"
	case sub.status == statusPolling:
		status = "polling"
	case sub.status == statusFailedToPoll:
		status = "poll failed"
	case sub.status == statusSubNotReady:
		status = "sub not ready"
	case sub.status == statusImageNotReady:
		status = "image not ready"
	case sub.status == statusFetching:
		status = "fetching"
	case sub.status == statusFailedToFetch:
		status = "fetch failed"
	case sub.status == statusWaitingForNextPoll:
		status = "waiting for next poll"
	case sub.status == statusUpdating:
		status = "updating"
	case sub.status == statusFailedToUpdate:
		status = "update failed"
	case sub.status == statusSynced:
		status = "synced"
	}
	fmt.Fprintf(writer, "    <td>%s</td>\n", status)
	if sub.lastSuccessfulPoll.IsZero() {
		fmt.Fprintf(writer, "    <td></td>\n")
	} else {
		fmt.Fprintf(writer, "    <td>%s</td>\n",
			time.Since(sub.lastSuccessfulPoll))
	}
	fmt.Fprintf(writer, "  </tr>\n")
}
