package metadatad

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/Symantec/Dominator/lib/json"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
)

func (s *server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	hostname, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		fmt.Fprintln(w, err)
		return
	}
	ipAddr := net.ParseIP(hostname)
	if rawHandler, ok := s.rawHandlers[req.URL.Path]; ok {
		rawHandler(w, ipAddr)
		return
	}
	vmInfo, err := s.manager.GetVmInfo(ipAddr)
	if err != nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		return
	}
	infoHandler, ok := s.infoHandlers[req.URL.Path]
	if !ok {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		return
	}
	writer := bufio.NewWriter(w)
	defer writer.Flush()
	if err := infoHandler(writer, vmInfo); err != nil {
		fmt.Fprintln(writer, err)
	}
}

func (s *server) showTime(writer io.Writer, vmInfo proto.VmInfo) error {
	now := time.Now()
	nano := now.UnixNano() - now.Unix()*1000000000
	_, err := fmt.Fprintf(writer, "%d.%09d\n", now.Unix(), nano)
	return err
}

func (s *server) showVM(writer io.Writer, vmInfo proto.VmInfo) error {
	return json.WriteWithIndent(writer, "    ", vmInfo)
}

func (s *server) showUserData(w http.ResponseWriter, ipAddr net.IP) {
	if file, err := s.manager.GetVmUserData(ipAddr); err != nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
	} else {
		defer file.Close()
		writer := bufio.NewWriter(w)
		defer writer.Flush()
		io.Copy(writer, file)
	}
}
