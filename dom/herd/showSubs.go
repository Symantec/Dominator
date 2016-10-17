package herd

import (
	"fmt"
	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/format"
	"github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/lib/url"
	"io"
	"net/http"
	"strings"
	"time"
)

func (herd *Herd) showAliveSubsHandler(w io.Writer, req *http.Request) {
	herd.showSubs(w, "alive ", selectAliveSub)
}

func (herd *Herd) showAllSubsHandler(w io.Writer, req *http.Request) {
	herd.showSubs(w, "", nil)
}

func (herd *Herd) showCompliantSubsHandler(w io.Writer, req *http.Request) {
	herd.showSubs(w, "compliant ", selectCompliantSub)
}

func (herd *Herd) showDeviantSubsHandler(w io.Writer, req *http.Request) {
	herd.showSubs(w, "deviant ", selectDeviantSub)
}

func (herd *Herd) showReachableSubsHandler(w io.Writer, req *http.Request) {
	selector, err := herd.getReachableSelector(url.ParseQuery(req.URL))
	if err != nil {
		fmt.Fprintln(w, err)
		return
	}
	herd.showSubs(w, "reachable ", selector)
}

func (herd *Herd) showSubs(writer io.Writer, subType string,
	selectFunc func(*Sub) bool) {
	fmt.Fprintf(writer, "<title>Dominator %s subs</title>", subType)
	fmt.Fprintln(writer, `<style>
                          table, th, td {
                          border-collapse: collapse;
                          }
                          </style>`)
	if srpc.CheckTlsRequired() {
		fmt.Fprintln(writer, "<body>")
	} else {
		fmt.Fprintln(writer, "<body bgcolor=\"#ffb0b0\">")
		fmt.Fprintln(writer,
			`<h1><center><font color="red">Running in insecure mode. You can get pwned!!!</center></font></h1>`)
	}
	if herd.updatesDisabledReason != "" {
		fmt.Fprintf(writer, "<center>")
		herd.writeDisableStatus(writer)
		fmt.Fprintln(writer, "</center>")
	}
	fmt.Fprintln(writer, `<table border="1" style="width:100%">`)
	fmt.Fprintln(writer, "  <tr>")
	fmt.Fprintln(writer, "    <th>Name</th>")
	fmt.Fprintln(writer, "    <th>Required Image</th>")
	fmt.Fprintln(writer, "    <th>Planned Image</th>")
	fmt.Fprintln(writer, "    <th>Busy</th>")
	fmt.Fprintln(writer, "    <th>Status</th>")
	fmt.Fprintln(writer, "    <th>Uptime</th>")
	fmt.Fprintln(writer, "    <th>Last Scan Duration</th>")
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
}

func showSub(writer io.Writer, sub *Sub) {
	if sub.isInsecure {
		fmt.Fprintln(writer, "  <tr style=\"background-color:yellow\">")
	} else {
		fmt.Fprintln(writer, "  <tr>")
	}
	subURL := fmt.Sprintf("http://%s:%d/",
		strings.SplitN(sub.String(), "*", 2)[0], constants.SubPortNumber)
	fmt.Fprintf(writer, "    <td><a href=\"%s\">%s</a></td>\n", subURL, sub)
	sub.herd.showImage(writer, sub.mdb.RequiredImage, true)
	sub.herd.showImage(writer, sub.mdb.PlannedImage, false)
	sub.showBusy(writer)
	fmt.Fprintf(writer, "    <td><a href=\"showSub?%s\">%s</a></td>\n",
		sub.mdb.Hostname, sub.publishedStatus.html())
	timeNow := time.Now()
	showSince(writer, sub.pollTime, sub.startTime)
	showDuration(writer, sub.lastScanDuration, false)
	showSince(writer, timeNow, sub.lastPollSucceededTime)
	showSince(writer, timeNow, sub.lastUpdateTime)
	showSince(writer, timeNow, sub.lastSyncTime)
	showDuration(writer, sub.lastConnectDuration, false)
	showDuration(writer, sub.lastShortPollDuration, !sub.lastPollWasFull)
	showDuration(writer, sub.lastFullPollDuration, sub.lastPollWasFull)
	showDuration(writer, sub.lastComputeUpdateCpuDuration, false)
	fmt.Fprintln(writer, "  </tr>")
}

func (herd *Herd) showImage(writer io.Writer, name string, showDefault bool) {
	if name == "" {
		if showDefault && herd.defaultImageName != "" {
			fmt.Fprintf(writer,
				"    <td><a style=\"color: #CCCC00\" href=\"http://%s/showImage?%s\">%s</a></td>\n",
				herd.imageManager, herd.defaultImageName, herd.defaultImageName)
		} else {
			fmt.Fprintln(writer, "    <td></td>")
		}
	} else if image, err := herd.imageManager.Get(name, false); err != nil {
		fmt.Fprintf(writer, "    <td><font color=\"red\">%s</font></td>\n", err)
	} else if image != nil {
		fmt.Fprintf(writer,
			"    <td><a href=\"http://%s/showImage?%s\">%s</a></td>\n",
			herd.imageManager, name, name)
	} else {
		fmt.Fprintf(writer, "    <td><font color=\"grey\">%s</font></td>\n",
			name)
	}
}

func (herd *Herd) showSubHandler(w io.Writer, req *http.Request) {
	subName := req.URL.RawQuery
	fmt.Fprintf(w, "<title>sub %s</title>", subName)
	if srpc.CheckTlsRequired() {
		fmt.Fprintln(w, "<body>")
	} else {
		fmt.Fprintln(w, "<body bgcolor=\"#ffb0b0\">")
		fmt.Fprintln(w,
			`<h1><center><font color="red">Running in insecure mode. You can get pwned!!!</center></font></h1>`)
	}
	if herd.updatesDisabledReason != "" {
		fmt.Fprintf(w, "<center>")
		herd.writeDisableStatus(w)
		fmt.Fprintln(w, "</center>")
	}
	fmt.Fprintln(w, "<h3>")
	sub := herd.getSub(subName)
	if sub == nil {
		fmt.Fprintf(w, "Sub: %s UNKNOWN!\n", subName)
		return
	}
	timeNow := time.Now()
	subURL := fmt.Sprintf("http://%s:%d/",
		strings.SplitN(sub.String(), "*", 2)[0], constants.SubPortNumber)
	fmt.Fprintf(w, "Information for sub: <a href=\"%s\">%s</a><br>\n",
		subURL, subName)
	fmt.Fprintln(w, "</h3>")
	fmt.Fprint(w, "<table border=\"0\">\n")
	newRow(w, "Required Image", true)
	sub.herd.showImage(w, sub.mdb.RequiredImage, true)
	newRow(w, "Planned Image", false)
	sub.herd.showImage(w, sub.mdb.PlannedImage, false)
	newRow(w, "Busy time", false)
	sub.showBusy(w)
	newRow(w, "Status", false)
	fmt.Fprintf(w, "    <td>%s</td>\n", sub.publishedStatus.html())
	newRow(w, "Uptime", false)
	showSince(w, sub.pollTime, sub.startTime)
	newRow(w, "Last scan duration", false)
	showDuration(w, sub.lastScanDuration, false)
	newRow(w, "Time since last successful poll", false)
	showSince(w, timeNow, sub.lastPollSucceededTime)
	newRow(w, "Time since last update", false)
	showSince(w, timeNow, sub.lastUpdateTime)
	newRow(w, "Time since last sync", false)
	showSince(w, timeNow, sub.lastSyncTime)
	newRow(w, "Last connection duration", false)
	showDuration(w, sub.lastConnectDuration, false)
	newRow(w, "Last short poll duration", false)
	showDuration(w, sub.lastShortPollDuration, !sub.lastPollWasFull)
	newRow(w, "Last full poll duration", false)
	showDuration(w, sub.lastFullPollDuration, sub.lastPollWasFull)
	newRow(w, "Last compute duration", false)
	showDuration(w, sub.lastComputeUpdateCpuDuration, false)
	fmt.Fprint(w, "  </tr>\n")
	fmt.Fprint(w, "</table>\n")
	fmt.Fprintln(w, "MDB Data:")
	fmt.Fprintln(w, "<pre>")
	json.WriteWithIndent(w, "    ", sub.mdb)
	fmt.Fprintln(w, "</pre>")
}

func newRow(w io.Writer, row string, first bool) {
	if !first {
		fmt.Fprint(w, "  </tr>\n")
	}
	fmt.Fprint(w, "  <tr>\n")
	fmt.Fprintf(w, "    <td>%s:</td>\n", row)
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
		fmt.Fprintln(writer, "    <td></td>")
	} else {
		showDuration(writer, now.Sub(since), false)
	}
}

func showDuration(writer io.Writer, duration time.Duration, highlight bool) {
	if duration < 1 {
		fmt.Fprintf(writer, "    <td></td>\n")
	} else {
		str := format.Duration(duration)
		if highlight {
			str = "<b>" + str + "</b>"
		}
		fmt.Fprintf(writer, "    <td>%s</td>\n", str)
	}
}
