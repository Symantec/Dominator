package logbuf

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"
)

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
	tmpNames := make([]string, 0, len(names))
	for _, name := range names {
		if strings.Index(name, ":") >= 0 {
			tmpNames = append(tmpNames, name)

		}
	}
	names = tmpNames
	sort.Strings(names)
	flags, _ := parseQuery(req.URL.RawQuery)
	recentFirstString := ""
	_, recentFirst := flags["recentFirst"]
	if recentFirst {
		recentFirstString = "&recentFirst"
		reverseStrings(names)
	}
	if _, ok := flags["text"]; ok {
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
	// TODO(rgooch): Enable when ready.
	// showRecentLinks(writer, recentFirstString)
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

func showRecentLinks(writer io.Writer, recentFirstString string) {
	fmt.Fprintln(writer, `Show last: <a href="logs/showLast?1m%s">min</a>`,
		recentFirstString)
	fmt.Fprintln(writer, `           <a href="logs/showLast?10m%s">10 min</a>`,
		recentFirstString)
	fmt.Fprintln(writer, `           <a href="logs/showLast?1h%s">hour</a>`,
		recentFirstString)
	fmt.Fprintln(writer, `           <a href="logs/showLast?1d%s">day</a>`,
		recentFirstString)
}

func (lb *LogBuffer) httpDumpHandler(w http.ResponseWriter, req *http.Request) {
	flags, pairs := parseQuery(req.URL.RawQuery)
	name, ok := pairs["name"]
	if !ok {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	recentFirst := false
	if _, ok := flags["recentFirst"]; ok {
		recentFirst = true
	}
	if name == "latest" {
		writer := bufio.NewWriter(w)
		defer writer.Flush()
		lb.Dump(writer, "", "", recentFirst)
		return
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
	flags, _ := parseQuery(req.URL.RawQuery)
	_, recentFirst := flags["recentFirst"]
	for flag := range flags {
		length := len(flag)
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
		default:
			continue
		}
		if val, err := strconv.ParseUint(flag[:length-1], 10, 64); err != nil {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			return
		} else {
			lb.showRecent(w, time.Duration(val)*unit, recentFirst)
		}
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusBadRequest)
}

func (lb *LogBuffer) showRecent(w io.Writer, duration time.Duration,
	recentFirst bool) {
}

func (lb *LogBuffer) writeHtml(writer io.Writer) {
	fmt.Fprintln(writer, `<a href="logs">Logs:</a><br>`)
	fmt.Fprintln(writer, "<pre>")
	lb.Dump(writer, "", "", false)
	fmt.Fprintln(writer, "</pre>")
}

func parseQuery(rawQuery string) (map[string]struct{}, map[string]string) {
	flags := make(map[string]struct{})
	table := make(map[string]string)
	for _, pair := range strings.Split(rawQuery, "&") {
		splitPair := strings.Split(pair, "=")
		if len(splitPair) == 1 {
			flags[splitPair[0]] = struct{}{}
		}
		if len(splitPair) == 2 {
			table[splitPair[0]] = splitPair[1]
		}
	}
	return flags, table
}
