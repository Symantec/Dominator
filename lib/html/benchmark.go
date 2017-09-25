package html

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"time"

	"github.com/Symantec/Dominator/lib/wsyscall"
)

func benchmarkedHandler(handler func(io.Writer,
	*http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		runtime.LockOSThread()
		var startRusage, stopRusage wsyscall.Rusage
		wsyscall.Getrusage(wsyscall.RUSAGE_THREAD, &startRusage)
		startTime := time.Now()
		writer := bufio.NewWriter(w)
		defer writer.Flush()
		defer fmt.Fprintln(writer, "</body>")
		handler(writer, req)
		durationReal := time.Since(startTime)
		err := wsyscall.Getrusage(wsyscall.RUSAGE_THREAD, &stopRusage)
		if err != nil {
			fmt.Fprintf(writer,
				"<br><font color=\"grey\">Render time: real: %s  wbuf: %d B</font>\n",
				durationReal, writer.Buffered())
			return
		}
		var durationUser, durationSys int64
		durationUser = (stopRusage.Utime.Sec - startRusage.Utime.Sec) * 1000000
		durationUser += stopRusage.Utime.Usec - startRusage.Utime.Usec
		durationSys = (stopRusage.Stime.Sec - startRusage.Stime.Sec) * 1000000
		durationSys += stopRusage.Stime.Usec - startRusage.Stime.Usec
		fmt.Fprintf(writer,
			"<br><font color=\"grey\">Render time: real: %s, user: %d us, sys: %d us  wbuf: %d B</font>\n",
			durationReal, durationUser, durationSys, writer.Buffered())
	}
}
