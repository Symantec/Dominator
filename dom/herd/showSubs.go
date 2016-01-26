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
	fmt.Fprintln(writer, "    <th>Uptime</th>")
	fmt.Fprintln(writer, "    <th>Staleness</th>")
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
		strings.SplitN(sub.hostname, "*", 2)[0], constants.SubPortNumber)
	fmt.Fprintf(writer, "    <td><a href=\"%s\">%s</a></td>\n",
		subURL, sub.hostname)
	sub.herd.showImage(writer, sub.requiredImage)
	sub.herd.showImage(writer, sub.plannedImage)
	fmt.Fprintf(writer, "    <td>%v</td>\n", sub.busy)
	fmt.Fprintf(writer, "    <td>%s</td>\n", sub.status)
	if sub.startTime.IsZero() || sub.pollTime.IsZero() {
		fmt.Fprintf(writer, "    <td></td>\n")
	} else {
		fmt.Fprintf(writer, "    <td>%s</td>\n", sub.pollTime.Sub(sub.startTime))
	}
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

func (herd *Herd) showImage(writer io.Writer, name string) {
	if image, err := herd.getImage(name); image != nil {
		fmt.Fprintf(writer, "    <td>%s</td>\n", name)
	} else if err != nil {
		fmt.Fprintf(writer, "    <td><font color=\"red\">%s</font></td>\n", err)
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
