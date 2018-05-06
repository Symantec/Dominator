package metadatad

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/Symantec/Dominator/lib/json"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
)

func (s *server) computePaths() {
	s.paths = make(map[string]struct{})
	for path := range s.infoHandlers {
		s.paths[path] = struct{}{}
	}
	for path := range s.rawHandlers {
		s.paths[path] = struct{}{}
	}
}

func (s *server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	hostname, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		fmt.Fprintln(w, err)
		return
	}
	ipAddr := net.ParseIP(hostname)
	s.manager.NotifyVmMetadataRequest(ipAddr, req.URL.Path)
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
	writer := bufio.NewWriter(w)
	defer writer.Flush()
	if infoHandler, ok := s.infoHandlers[req.URL.Path]; ok {
		if err := infoHandler(writer, vmInfo); err != nil {
			fmt.Fprintln(writer, err)
		}
		return
	}
	paths := make([]string, 0)
	pathsSet := make(map[string]struct{})
	for path := range s.paths {
		if strings.HasPrefix(path, req.URL.Path) {
			splitPath := strings.Split(path[len(req.URL.Path):], "/")
			result := splitPath[0]
			if result == "" {
				result = splitPath[1]
			}
			if _, ok := pathsSet[result]; !ok {
				pathsSet[result] = struct{}{}
				paths = append(paths, result)
			}
		}
	}
	if len(paths) < 1 {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		return
	}
	sort.Strings(paths)
	for _, path := range paths {
		fmt.Fprintln(writer, path)
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

func (s *server) showSmallStack(w http.ResponseWriter, ipAddr net.IP) {
	w.Write([]byte("true\n"))
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
