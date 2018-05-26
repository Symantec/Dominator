package httpd

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"

	"github.com/Symantec/Dominator/fleetmanager/topology"
	"github.com/Symantec/Dominator/lib/log"
)

type HtmlWriter interface {
	WriteHtml(writer io.Writer)
}

type Server struct {
	htmlWriters []HtmlWriter
	logger      log.DebugLogger
	mutex       sync.RWMutex
	_topology   *topology.Topology
}

func StartServer(portNum uint, logger log.DebugLogger) (*Server, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", portNum))
	if err != nil {
		return nil, err
	}
	server := &Server{logger: logger}
	http.HandleFunc("/", server.statusHandler)
	go http.Serve(listener, nil)
	return server, nil
}

func (s *Server) AddHtmlWriter(htmlWriter HtmlWriter) {
	s.htmlWriters = append(s.htmlWriters, htmlWriter)
}

func (s *Server) UpdateTopology(t *topology.Topology) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s._topology = t
}
