package herd

import (
	"bufio"
	"fmt"
	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/format"
	"io"
	"net/http"
	"strings"
	"time"
)

func showAliveSubsHandler(w http.ResponseWriter, req *http.Request) {
	httpdHerd.showSubs(w, "alive ", selectAliveSub)
}

func showAllSubsHandler(w http.ResponseWriter, req *http.Request) {
	httpdHerd.showSubs(w, "", nil)
}

func showCompliantSubsHandler(w http.ResponseWriter, req *http.Request) {
	httpdHerd.showSubs(w, "compliant ", selectCompliantSub)
}

func showDeviantSubsHandler(w http.ResponseWriter, req *http.Request) {
	httpdHerd.showSubs(w, "deviant ", selectDeviantSub)
}

func showReachableSubsHandler(w http.ResponseWriter, req *http.Request) {
	var duration rDuration
	switch req.URL.RawQuery {
	case "1m":
		duration = rDuration(time.Second * 60)
	case "10m":
		duration = rDuration(time.Second * 600)
	case "1h":
		duration = rDuration(time.Second * 3600)
	case "1d":
		duration = rDuration(time.Second * 3600 * 24)
	}
	httpdHerd.showSubs(w, "reachable ", duration.selector)
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
	fmt.Fprintln(writer, "    <th>Uptime</th>")
	fmt.Fprintln(writer, "    <th>Staleness</th>")
	fmt.Fprintln(writer, "    <th>Last Update</th>")
	fmt.Fprintln(writer, "    <th>Last Sync</th>")
	fmt.Fprintln(writer, "    <th>Connect</th>")
	fmt.Fprintln(writer, "    <th>Short Poll</th>")
	fmt.Fprintln(writer, "    <th>Full Poll</th>")
	fmt.Fprintln(writer, "    <th>Update Compute</th>")
	fmt.Fprintln(writer, "  </tr>")
	subs := herd.getSelectedSubs(selectFunc)
	for _, sub := range subs {
		showSub(writer, sub)
	}
	fmt.Fprintln(writer, "</table>")
	fmt.Fprintln(writer, "</body>")
}

func showSub(writer io.Writer, sub *Sub) {
	fmt.Fprintf(writer, "  <tr>\n")
	subURL := fmt.Sprintf("http://%s:%d/",
		strings.SplitN(sub.String(), "*", 2)[0], constants.SubPortNumber)
	fmt.Fprintf(writer, "    <td><a href=\"%s\">%s</a></td>\n", subURL, sub)
	sub.herd.showImage(writer, sub.mdb.RequiredImage)
	sub.herd.showImage(writer, sub.mdb.PlannedImage)
	sub.showBusy(writer)
	fmt.Fprintf(writer, "    <td>%s</td>\n", sub.status)
	timeNow := time.Now()
	showSince(writer, sub.pollTime, sub.startTime)
	showSince(writer, timeNow, sub.lastPollSucceededTime)
	showSince(writer, timeNow, sub.lastUpdateTime)
	showSince(writer, timeNow, sub.lastSyncTime)
	showDuration(writer, sub.lastConnectDuration)
	showDuration(writer, sub.lastShortPollDuration)
	showDuration(writer, sub.lastFullPollDuration)
	showDuration(writer, sub.lastComputeUpdateCpuDuration)
	fmt.Fprintf(writer, "  </tr>\n")
}

func (herd *Herd) showImage(writer io.Writer, name string) {
	if name == "" {
		fmt.Fprintln(writer, "    <td></td>")
	} else if image, err := herd.getImage(name); err != nil {
		fmt.Fprintf(writer, "    <td><font color=\"red\">%s</font></td>\n", err)
	} else if image != nil {
		fmt.Fprintf(writer,
			"    <td><a href=\"http://%s/showImage?%s\">%s</a></td>\n",
			herd.imageServerAddress, name, name)
	} else {
		fmt.Fprintf(writer, "    <td><font color=\"grey\">%s</font></td>\n",
			name)
	}
}

func (sub *Sub) showBusy(writer io.Writer) {
	if sub.busy {
		if sub.busyStartTime.IsZero() {
			fmt.Fprintln(writer, "    <td>busy</td>")
		} else {
			fmt.Fprintf(writer, "    <td>%s</td>\n",
				format.Duration(time.Since(sub.busyStartTime)))
		}
	} else {
		if sub.busyStartTime.IsZero() {
			fmt.Fprintln(writer, "    <td></td>")
		} else {
			fmt.Fprintf(writer, "    <td><font color=\"grey\">%s</font></td>\n",
				format.Duration(sub.busyStopTime.Sub(sub.busyStartTime)))
		}
	}
}

func showSince(writer io.Writer, now time.Time, since time.Time) {
	if now.IsZero() || since.IsZero() {
		fmt.Fprintf(writer, "    <td></td>\n")
	} else {
		showDuration(writer, now.Sub(since))
	}
}

func showDuration(writer io.Writer, duration time.Duration) {
	if duration < 1 {
		fmt.Fprintf(writer, "    <td></td>\n")
	} else {
		fmt.Fprintf(writer, "    <td>%s</td>\n", format.Duration(duration))
	}
}
