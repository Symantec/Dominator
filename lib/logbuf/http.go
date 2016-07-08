package logbuf

import (
	"bufio"
	"fmt"
	"github.com/Symantec/Dominator/lib/url"
	"io"
	"net/http"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"
)

type countingWriter struct {
	count      uint64
	writer     io.Writer
	prefixLine string
}

func (w *countingWriter) Write(p []byte) (n int, err error) {
	if w.prefixLine != "" {
		w.writer.Write([]byte(w.prefixLine))
		w.prefixLine = ""
	}
	n, err = w.writer.Write(p)
	if n > 0 {
		w.count += uint64(n)
	}
	return
}

func (lb *LogBuffer) addHttpHandlers() {
	http.HandleFunc("/logs", lb.httpListHandler)
	http.HandleFunc("/logs/dump", lb.httpDumpHandler)
	http.HandleFunc("/logs/showLast", lb.httpShowLastHandler)
}

func (lb *LogBuffer) httpListHandler(w http.ResponseWriter, req *http.Request) {
	if lb.logDir == "" {
		return
	}
	writer := bufio.NewWriter(w)
	defer writer.Flush()
	parsedQuery := url.ParseQuery(req.URL)
	_, recentFirst := parsedQuery.Flags["recentFirst"]
	names, err := lb.list(recentFirst)
	if err != nil {
		fmt.Fprintln(writer, err)
		return
	}
	recentFirstString := ""
	if recentFirst {
		recentFirstString = "&recentFirst"
	}
	if parsedQuery.OutputType() == url.OutputTypeText {
		for _, name := range names {
			fmt.Fprintln(writer, name)
		}
		return
	}
	fmt.Fprintln(writer, "<body>")
	fmt.Fprint(writer, "Logs: ")
	if recentFirst {
		fmt.Fprintf(writer, "showing recent first ")
		fmt.Fprintln(writer, `<a href="logs">show recent last</a><br>`)
	} else {
		fmt.Fprintf(writer, "showing recent last ")
		fmt.Fprintln(writer,
			`<a href="logs?recentFirst">show recent first</a><br>`)
	}
	showRecentLinks(writer, recentFirstString)
	fmt.Fprintln(writer, "<p>")
	currentName := ""
	lb.rwMutex.Lock()
	if lb.file != nil {
		currentName = path.Base(lb.file.Name())
	}
	lb.rwMutex.Unlock()
	for _, name := range names {
		if name == currentName {
			fmt.Fprintf(writer,
				"<a href=\"logs/dump?name=latest%s\">%s</a> (current)<br>\n",
				recentFirstString, name)
		} else {
			fmt.Fprintf(writer, "<a href=\"logs/dump?name=%s%s\">%s</a><br>\n",
				name, recentFirstString, name)
		}
	}
	fmt.Fprintln(writer, "</body>")
}

func showRecentLinks(w io.Writer, recentFirstString string) {
	fmt.Fprintf(w, "Show last: <a href=\"logs/showLast?1m%s\">minute</a>\n",
		recentFirstString)
	fmt.Fprintf(w, "           <a href=\"logs/showLast?10m%s\">10 min</a>\n",
		recentFirstString)
	fmt.Fprintf(w, "           <a href=\"logs/showLast?1h%s\">hour</a>\n",
		recentFirstString)
	fmt.Fprintf(w, "           <a href=\"logs/showLast?1d%s\">day</a>\n",
		recentFirstString)
	fmt.Fprintf(w, "           <a href=\"logs/showLast?1w%s\">week</a>\n",
		recentFirstString)
}

func (lb *LogBuffer) httpDumpHandler(w http.ResponseWriter, req *http.Request) {
	parsedQuery := url.ParseQuery(req.URL)
	name, ok := parsedQuery.Table["name"]
	if !ok {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	_, recentFirst := parsedQuery.Flags["recentFirst"]
	if name == "latest" {
		lbFilename := ""
		lb.rwMutex.Lock()
		if lb.file != nil {
			lbFilename = lb.file.Name()
		}
		lb.rwMutex.Unlock()
		if lbFilename == "" {
			writer := bufio.NewWriter(w)
			defer writer.Flush()
			lb.Dump(writer, "", "", recentFirst)
			return
		}
		name = path.Base(lbFilename)
	}
	file, err := os.Open(path.Join(lb.logDir, path.Base(path.Clean(name))))
	if err != nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		return
	}
	defer file.Close()
	writer := bufio.NewWriter(w)
	defer writer.Flush()
	if recentFirst {
		scanner := bufio.NewScanner(file)
		lines := make([]string, 0)
		for scanner.Scan() {
			line := scanner.Text()
			if len(line) < 1 {
				continue
			}
			lines = append(lines, line)
		}
		if err = scanner.Err(); err == nil {
			reverseStrings(lines)
			for _, line := range lines {
				fmt.Fprintln(writer, line)
			}
		}
	} else {
		_, err = io.Copy(writer, bufio.NewReader(file))
	}
	if err != nil {
		fmt.Fprintln(writer, err)
	}
	return
}

func (lb *LogBuffer) httpShowLastHandler(w http.ResponseWriter,
	req *http.Request) {
	parsedQuery := url.ParseQuery(req.URL)
	_, recentFirst := parsedQuery.Flags["recentFirst"]
	for flag := range parsedQuery.Flags {
		length := len(flag)
		if length < 2 {
			continue
		}
		unitChar := flag[length-1]
		var unit time.Duration
		switch unitChar {
		case 's':
			unit = time.Second
		case 'm':
			unit = time.Minute
		case 'h':
			unit = time.Hour
		case 'd':
			unit = time.Hour * 24
		case 'w':
			unit = time.Hour * 24 * 7
		default:
			continue
		}
		if val, err := strconv.ParseUint(flag[:length-1], 10, 64); err != nil {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			return
		} else {
			lb.showRecent(w, time.Duration(val)*unit, recentFirst)
			return
		}
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusBadRequest)
}

func (lb *LogBuffer) showRecent(w io.Writer, duration time.Duration,
	recentFirst bool) {
	writer := bufio.NewWriter(w)
	defer writer.Flush()
	names, err := lb.list(true)
	if err != nil {
		fmt.Fprintln(writer, err)
		return
	}
	earliestTime := time.Now().Add(-duration)
	// Get a list of names which may be recent enough.
	tmpNames := make([]string, 0, len(names))
	for _, name := range names {
		startTime, err := time.ParseInLocation(timeLayout, name, time.Local)
		if err != nil {
			continue
		}
		tmpNames = append(tmpNames, name)
		if startTime.Before(earliestTime) {
			break
		}
	}
	names = tmpNames
	if !recentFirst {
		reverseStrings(names)
	}
	fmt.Fprintln(writer, "<body>")
	cWriter := &countingWriter{writer: writer}
	lb.rwMutex.Lock()
	lb.writer.Flush()
	lb.rwMutex.Unlock()
	for _, name := range names {
		cWriter.count = 0
		lb.dumpSince(cWriter, name, earliestTime, "", "<br>\n", recentFirst)
		if cWriter.count > 0 {
			cWriter.prefixLine = "<hr>\n"
		}
	}
	fmt.Fprintln(writer, "</body>")
}

func (lb *LogBuffer) list(recentFirst bool) ([]string, error) {
	file, err := os.Open(lb.logDir)
	if err != nil {
		return nil, err
	}
	names, err := file.Readdirnames(-1)
	file.Close()
	if err != nil {
		return nil, err
	}
	tmpNames := make([]string, 0, len(names))
	for _, name := range names {
		if strings.Count(name, ":") == 3 {
			tmpNames = append(tmpNames, name)
		}
	}
	names = tmpNames
	sort.Strings(names)
	if recentFirst {
		reverseStrings(names)
	}
	return names, nil
}

func (lb *LogBuffer) writeHtml(writer io.Writer) {
	fmt.Fprintln(writer, `<a href="logs">Logs:</a><br>`)
	fmt.Fprintln(writer, "<pre>")
	lb.Dump(writer, "", "", false)
	fmt.Fprintln(writer, "</pre>")
}
