package httpd

import (
	"bufio"
	"net"
	"net/http"

	"github.com/Cloud-Foundations/Dominator/lib/url"
)

func (s state) showBootLogHandler(w http.ResponseWriter, req *http.Request) {
	parsedQuery := url.ParseQuery(req.URL)
	if len(parsedQuery.Flags) != 1 {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var ipAddr string
	for name := range parsedQuery.Flags {
		ipAddr = name
	}
	r, err := s.manager.GetVmBootLog(net.ParseIP(ipAddr))
	if err != nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		return
	}
	defer r.Close()
	reader := bufio.NewReader(r)
	writer := bufio.NewWriter(w)
	defer writer.Flush()
	// Strip out useless and annoying CRLF and replace with LF only.
	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 1 {
			if line[len(line)-1] == '\n' && line[len(line)-2] == '\r' {
				line[len(line)-2] = '\n'
				line = line[:len(line)-1]
			}
			writer.Write(line)
		}
		if err != nil {
			return
		}
	}
}
