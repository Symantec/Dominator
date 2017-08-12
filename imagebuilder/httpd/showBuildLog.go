package httpd

import (
	"bufio"
	"fmt"
	"net/http"
)

func (s state) showCurrentBuildLogHandler(w http.ResponseWriter,
	req *http.Request) {
	writer := bufio.NewWriter(w)
	defer writer.Flush()
	streamName := req.URL.RawQuery
	fmt.Fprintf(writer, "<title>build log for stream %s</title>\n", streamName)
	fmt.Fprintln(writer, `<head><meta http-equiv="refresh" content="2"></head>`)
	fmt.Fprintln(writer, "<body>")
	fmt.Fprintln(writer, "<h3>")
	buildLog, err := s.builder.GetCurrentBuildLog(streamName)
	if err != nil {
		fmt.Fprintln(writer, err)
	} else if buildLog == nil {
		fmt.Fprintln(writer, "No build log")
	} else {
		fmt.Fprintf(writer, "In progress build log for stream: %s\n",
			streamName)
		fmt.Fprintln(writer, "</h3>")
		fmt.Fprintln(writer, "<pre>")
		writer.Write(buildLog)
		fmt.Fprintln(writer, "</pre>")
	}
	fmt.Fprintln(writer, `<a name="bottom"></a>`)
	fmt.Fprintln(writer, "</body>")
}

func (s state) showLastBuildLogHandler(w http.ResponseWriter,
	req *http.Request) {
	writer := bufio.NewWriter(w)
	defer writer.Flush()
	streamName := req.URL.RawQuery
	fmt.Fprintf(writer, "<title>build log for stream %s</title>\n", streamName)
	fmt.Fprintln(writer, "<body>")
	fmt.Fprintln(writer, "<h3>")
	buildLog, err := s.builder.GetLatestBuildLog(streamName)
	if err != nil {
		fmt.Fprintln(writer, err)
	} else if buildLog == nil {
		fmt.Fprintln(writer, "No build log")
	} else {
		fmt.Fprintf(writer, "Lastest build log for stream: %s\n", streamName)
		fmt.Fprintln(writer, "</h3>")
		fmt.Fprintln(writer, "<pre>")
		writer.Write(buildLog)
		fmt.Fprintln(writer, "</pre>")
	}
	fmt.Fprintln(writer, "</body>")
}
