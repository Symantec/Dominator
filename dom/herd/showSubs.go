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

func showAllSubsHandler(w http.ResponseWriter, req *http.Request) {
	httpdHerd.showSubs(w, "", nil)
}

func showAliveSubsHandler(w http.ResponseWriter, req *http.Request) {
	httpdHerd.showSubs(w, "alive ", selectAliveSub)
}

func showDeviantSubsHandler(w http.ResponseWriter, req *http.Request) {
	httpdHerd.showSubs(w, "deviant ", selectDeviantSub)
}

func showCompliantSubsHandler(w http.ResponseWriter, req *http.Request) {
	httpdHerd.showSubs(w, "compliant ", selectCompliantSub)
}

func (herd *Herd) showSubs(w io.Writer, subType string,
	selectFunc func(*Sub) bool) {
	writer := bufio.NewWriter(w)
	defer writer.Flush()
	fmt.Fprintf(writer, "<title>Dominator %s subs</title>", subType)
	fmt.Fprintln(writer, `<style>
                          table, th, td {
                          border-collapse: collapse;
                          }
                          </style>`)
	fmt.Fprintln(writer, "<body>")
	fmt.Fprintln(writer, "<h3>")
	fmt.Fprintln(writer, `<table border="1" style="width:100%">`)
	fmt.Fprintln(writer, "  <tr>")
	fmt.Fprintln(writer, "    <th>Name</th>")
	fmt.Fprintln(writer, "    <th>Required Image</th>")
	fmt.Fprintln(writer, "    <th>Planned Image</th>")
	fmt.Fprintln(writer, "    <th>Busy</th>")
	fmt.Fprintln(writer, "    <th>Status</th>")
	fmt.Fprintln(writer, "    <th>Staleness</th>")
	fmt.Fprintln(writer, "    <th>Connect</th>")
	fmt.Fprintln(writer, "    <th>Short Poll</th>")
	fmt.Fprintln(writer, "    <th>Full Poll</th>")
	fmt.Fprintln(writer, "    <th>Update Compute</th>")
	fmt.Fprintln(writer, "  </tr>")
	subs := herd.getSelectedSubs(selectFunc)
	missingImages := make(map[string]struct{})
	for _, sub := range subs {
		showSub(writer, sub, missingImages)
	}
	fmt.Fprintln(writer, "</table>")
	fmt.Fprintln(writer, "</body>")
}

func showSub(writer io.Writer, sub *Sub, missingImages map[string]struct{}) {
	fmt.Fprintf(writer, "  <tr>\n")
	subURL := fmt.Sprintf("http://%s:%d/",
		strings.SplitN(sub.hostname, "*", 2)[0], constants.SubPortNumber)
	fmt.Fprintf(writer, "    <td><a href=\"%s\">%s</a></td>\n",
		subURL, sub.hostname)
	sub.herd.showImage(writer, sub.requiredImage, missingImages)
	sub.herd.showImage(writer, sub.plannedImage, missingImages)
	fmt.Fprintf(writer, "    <td>%v</td>\n", sub.busy)
	var status string
	switch {
	case sub.status == statusUnknown:
		status = "unknown"
	case sub.status == statusConnecting:
		status = "connecting"
	case sub.status == statusFailedToConnect:
		status = "connect failed"
	case sub.status == statusWaitingToPoll:
		status = "waiting to poll"
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
	case sub.status == statusComputingUpdate:
		status = "computing update"
	case sub.status == statusUpdating:
		status = "updating"
	case sub.status == statusFailedToUpdate:
		status = "update failed"
	case sub.status == statusSynced:
		status = "synced"
	default:
		panic(fmt.Sprintf("unknown status: %d", sub.status))
	}
	fmt.Fprintf(writer, "    <td>%s</td>\n", status)
	if sub.lastPollSucceededTime.IsZero() {
		fmt.Fprintf(writer, "    <td></td>\n")
	} else {
		fmt.Fprintf(writer, "    <td>%s</td>\n",
			time.Since(sub.lastPollSucceededTime))
	}
	showTime(writer, sub.lastConnectDuration)
	showTime(writer, sub.lastShortPollDuration)
	showTime(writer, sub.lastFullPollDuration)
	showTime(writer, sub.lastComputeUpdateCpuDuration)
	fmt.Fprintf(writer, "  </tr>\n")
}

func (herd *Herd) showImage(writer io.Writer, name string,
	missingImages map[string]struct{}) {
	found := false
	if _, ok := missingImages[name]; !ok {
		if herd.getImage(name) == nil {
			missingImages[name] = struct{}{}
		} else {
			found = true
		}
	}
	if found {
		fmt.Fprintf(writer, "    <td>%s</td>\n", name)
	} else {
		fmt.Fprintf(writer, "    <td><font color=\"grey\">%s</font></td>\n",
			name)
	}
}

func showTime(writer io.Writer, duration time.Duration) {
	if duration < 1 {
		fmt.Fprintf(writer, "    <td></td>\n")
	} else {
		seconds := duration.Seconds()
		if seconds <= 1.0 {
			fmt.Fprintf(writer, "    <td>%.fms</td>\n", seconds*1e3)
		} else {
			fmt.Fprintf(writer, "    <td>%.fs</td>\n", seconds)
		}
	}
}
