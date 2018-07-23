package tftpbootd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"strings"
	"time"

	imageclient "github.com/Symantec/Dominator/imageserver/client"
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/format"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/log/prefixlogger"
	objectclient "github.com/Symantec/Dominator/lib/objectserver/client"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/pin/tftp"
)

const tftpbootPrefix = "/tftpboot"

func cleanPath(filename string) string {
	if strings.HasPrefix(filename, tftpbootPrefix) {
		return filename[len(tftpbootPrefix):]
	} else if filename[0] != '/' {
		return "/" + filename
	} else {
		return filename
	}
}

func readHandler(rf io.ReaderFrom, reader io.Reader,
	logger log.DebugLogger) error {
	startTime := time.Now()
	nRead, err := rf.ReadFrom(reader)
	if err != nil {
		io.Copy(ioutil.Discard, reader)
		return err
	}
	timeTaken := time.Since(startTime)
	speed := uint64(float64(nRead) / timeTaken.Seconds())
	logger.Printf("%d bytes sent in %s (%s/s)\n",
		nRead, format.Duration(timeTaken), format.FormatBytes(speed))
	return nil
}

func newServer(imageServerAddress, imageStreamName string,
	logger log.DebugLogger) (*TftpbootServer, error) {
	s := &TftpbootServer{
		filesForIPs:        make(map[string]map[string][]byte),
		imageServerAddress: imageServerAddress,
		imageStreamName:    imageStreamName,
		logger:             logger,
		closeClientTimer:   time.NewTimer(time.Minute),
	}
	s.tftpdServer = tftp.NewServer(s.readHandler, nil)
	go func() {
		if err := s.tftpdServer.ListenAndServe(":69"); err != nil {
			s.logger.Println(err)
		}
	}()
	go s.imageServerClientCloser()
	return s, nil
}

func (s *TftpbootServer) closeImageServerClient() {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.imageServerClientInUse {
		return
	}
	if s.imageServerClient != nil {
		s.imageServerClient.Close()
		s.imageServerClient = nil
		s.logger.Debugf(0, "Closed connection to: %s\n", s.imageServerAddress)
	}
}

func (s *TftpbootServer) getImageServerClient() *srpc.Client {
	s.lock.Lock()
	s.imageServerClientInUse = true
	s.lock.Unlock()
	s.imageServerClientLock.Lock()
	if s.imageServerClient != nil {
		return s.imageServerClient
	}
	for ; ; time.Sleep(time.Second * 15) {
		client, err := srpc.DialHTTP("tcp", s.imageServerAddress, 0)
		if err != nil {
			s.logger.Println(err)
			continue
		}
		s.logger.Debugf(0, "Connected to: %s\n", s.imageServerAddress)
		s.imageServerClient = client
		return s.imageServerClient
	}
}

func (s *TftpbootServer) imageServerClientCloser() {
	for range s.closeClientTimer.C {
		s.closeImageServerClient()
	}
}

func (s *TftpbootServer) readHandler(filename string, rf io.ReaderFrom) error {
	filename = cleanPath(filename)
	rAddr := rf.(tftp.OutgoingTransfer).RemoteAddr().IP.String()
	logger := prefixlogger.New("tftpd("+rAddr+":"+filename+"): ", s.logger)
	if err := s.readHandlerInternal(filename, rf, rAddr, logger); err != nil {
		logger.Println(err)
		return err
	}
	return nil
}

func (s *TftpbootServer) readHandlerInternal(filename string, rf io.ReaderFrom,
	remoteAddr string, logger log.DebugLogger) error {
	s.lock.Lock()
	if files, ok := s.filesForIPs[remoteAddr]; ok {
		if data, ok := files[filename]; ok {
			s.lock.Unlock()
			rf.(tftp.OutgoingTransfer).SetSize(int64(len(data)))
			return readHandler(rf, bytes.NewReader(data), logger)
		}
	}
	imageStreamName := s.imageStreamName
	s.lock.Unlock()
	if imageStreamName == "" {
		return errors.New("no image stream defined")
	}
	client := s.getImageServerClient()
	defer s.releaseImageServerClient()
	imageName, err := imageclient.FindLatestImage(client, imageStreamName,
		false)
	if err != nil {
		return fmt.Errorf("error finding latest image in stream: %s: %s",
			imageStreamName, err)
	}
	if imageName == "" {
		return fmt.Errorf("no images in stream: %s", imageStreamName)
	}
	image, err := imageclient.GetImage(client, imageName)
	if err != nil {
		return fmt.Errorf("error getting image: %s: %s", imageName, err)
	}
	if err := image.FileSystem.RebuildInodePointers(); err != nil {
		return err
	}
	filenameToInodeTable := image.FileSystem.FilenameToInodeTable()
	if inum, ok := filenameToInodeTable[filename]; !ok {
		return os.ErrNotExist
	} else if gInode, ok := image.FileSystem.InodeTable[inum]; !ok {
		return fmt.Errorf("inode: %d does not exist", inum)
	} else if inode, ok := gInode.(*filesystem.RegularInode); !ok {
		return fmt.Errorf("inode is not a regular file: %d", inum)
	} else {
		objSrv := objectclient.AttachObjectClient(client)
		defer objSrv.Close()
		if size, reader, err := objSrv.GetObject(inode.Hash); err != nil {
			return err
		} else {
			defer reader.Close()
			rf.(tftp.OutgoingTransfer).SetSize(int64(size))
			return readHandler(rf, reader, logger)
		}
	}
}

func (s *TftpbootServer) registerFiles(ipAddr net.IP, files map[string][]byte) {
	address := ipAddr.String()
	cleanedFiles := make(map[string][]byte, len(files))
	for filename, data := range files {
		cleanedFiles[cleanPath(filename)] = data
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	if len(files) < 1 {
		delete(s.filesForIPs, address)
	} else {
		s.filesForIPs[address] = cleanedFiles
	}
}

func (s *TftpbootServer) releaseImageServerClient() {
	s.closeClientTimer.Reset(time.Minute)
	s.lock.Lock()
	s.imageServerClientInUse = false
	s.lock.Unlock()
	s.imageServerClientLock.Unlock()
}

func (s *TftpbootServer) setImageStreamName(name string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.imageStreamName = name
}

func (s *TftpbootServer) unregisterFiles(ipAddr net.IP) {
	address := ipAddr.String()
	s.lock.Lock()
	defer s.lock.Unlock()
	delete(s.filesForIPs, address)
}
