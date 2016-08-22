package html

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"syscall"
	"time"
)

func benchmarkedHandler(handler func(io.Writer,
	*http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		runtime.LockOSThread()
		var startRusage, stopRusage syscall.Rusage
		syscall.Getrusage(syscall.RUSAGE_THREAD, &startRusage)
		startTime := time.Now()
		writer := bufio.NewWriter(w)
		defer writer.Flush()
		handler(writer, req)
		durationReal := time.Since(startTime)
		syscall.Getrusage(syscall.RUSAGE_THREAD, &stopRusage)
		var durationUser, durationSys int64
		durationUser = (stopRusage.Utime.Sec - startRusage.Utime.Sec) * 1000000
		durationUser += stopRusage.Utime.Usec - startRusage.Utime.Usec
		durationSys = (stopRusage.Stime.Sec - startRusage.Stime.Sec) * 1000000
		durationSys += stopRusage.Stime.Usec - startRusage.Stime.Usec
		fmt.Fprintf(writer,
			"<br><font color=\"grey\">Render time: real: %s, user: %d us, sys: %d us  wbuf: %d B</font>\n",
			durationReal, durationUser, durationSys, writer.Buffered())
		fmt.Fprintln(writer, "</body>")
	}
}
