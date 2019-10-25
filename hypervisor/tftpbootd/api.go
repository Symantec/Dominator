package tftpbootd

import (
	"net"
	"sync"
	"time"

	"github.com/Cloud-Foundations/Dominator/lib/filesystem"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/pin/tftp"
)

type cachedFileSystem struct {
	deleteTimer *time.Timer
	fileSystem  *filesystem.FileSystem
}

type TftpbootServer struct {
	closeClientTimer       *time.Timer
	imageServerAddress     string
	logger                 log.DebugLogger
	tftpdServer            *tftp.Server
	lock                   sync.Mutex
	cachedFileSystems      map[string]*cachedFileSystem // Key: image stream.
	filesForIPs            map[string]map[string][]byte
	imageServerClientInUse bool
	imageStreamName        string
	imageServerClientLock  sync.Mutex
	imageServerClient      *srpc.Client
}

func New(imageServerAddress, imageStreamName string,
	logger log.DebugLogger) (*TftpbootServer, error) {
	return newServer(imageServerAddress, imageStreamName, logger)
}

func (s *TftpbootServer) RegisterFiles(ipAddr net.IP, files map[string][]byte) {
	s.registerFiles(ipAddr, files)
}

func (s *TftpbootServer) SetImageStreamName(name string) {
	s.setImageStreamName(name)
}

func (s *TftpbootServer) UnregisterFiles(ipAddr net.IP) {
	s.unregisterFiles(ipAddr)
}
