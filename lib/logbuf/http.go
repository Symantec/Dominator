package logbuf

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"sort"
)

func (lb *LogBuffer) addHttpHandlers() {
	http.HandleFunc("/logs", lb.httpListHandler)
	http.HandleFunc("/logs/dump", lb.httpDumpHandler)
}

func (lb *LogBuffer) httpListHandler(w http.ResponseWriter, req *http.Request) {
	if lb.logDir == "" {
		return
	}
	writer := bufio.NewWriter(w)
	defer writer.Flush()
	file, err := os.Open(lb.logDir)
	if err != nil {
		fmt.Fprintln(writer, err)
		return
	}
	names, err := file.Readdirnames(-1)
	file.Close()
	if err != nil {
		fmt.Fprintln(writer, err)
		return
	}
	sort.Strings(names)
	for _, name := range names {
		fmt.Fprintf(writer, "<a href=\"logs/dump?%s\">%s</a><br>\n",
			name, name)
	}
}

func (lb *LogBuffer) httpDumpHandler(w http.ResponseWriter, req *http.Request) {
	query := req.URL.RawQuery
	if query == "latest" {
		writer := bufio.NewWriter(w)
		defer writer.Flush()
		lb.Dump(writer, "", "")
		return
	}
	file, err := os.Open(path.Join(lb.logDir, path.Base(path.Clean(query))))
	if err != nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		return
	}
	defer file.Close()
	writer := bufio.NewWriter(w)
	defer writer.Flush()
	_, err = io.Copy(writer, bufio.NewReader(file))
	if err != nil {
		fmt.Fprintln(writer, err)
	}
	return
}

func (lb *LogBuffer) writeHtml(writer io.Writer) {
	fmt.Fprintln(writer, `<a href="logs">Logs:</a><br>`)
	fmt.Fprintln(writer, "<pre>")
	lb.Dump(writer, "", "")
	fmt.Fprintln(writer, "</pre>")
}
