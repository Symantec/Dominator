// +build linux

package main

import (
	"bufio"
	"fmt"
	"io"
	stdlog "log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"syscall"

	"github.com/Symantec/Dominator/lib/html"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/log/debuglogger"
	"github.com/Symantec/Dominator/lib/log/teelogger"
	"github.com/Symantec/Dominator/lib/srpc"
)

type HtmlWriter interface {
	WriteHtml(writer io.Writer)
}

type state struct {
	logger log.DebugLogger
}

type srpcType struct {
	remoteShellWaitGroup *sync.WaitGroup
	logger               log.DebugLogger
	mutex                sync.RWMutex
	connections          map[*srpc.Conn]struct{}
}

var htmlWriters []HtmlWriter

func startServer(portNum uint, remoteShellWaitGroup *sync.WaitGroup,
	logger log.DebugLogger) (log.DebugLogger, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", portNum))
	if err != nil {
		return nil, err
	}
	myState := state{logger}
	html.HandleFunc("/", myState.statusHandler)
	srpcObj := &srpcType{
		remoteShellWaitGroup: remoteShellWaitGroup,
		logger:               logger,
		connections:          make(map[*srpc.Conn]struct{}),
	}
	if err := srpc.RegisterName("Installer", srpcObj); err != nil {
		logger.Printf("error registering SRPC receiver: %s\n", err)
	}
	sprayLogger := debuglogger.New(stdlog.New(srpcObj, "", 0))
	sprayLogger.SetLevel(int16(*logDebugLevel))
	go http.Serve(listener, nil)
	return teelogger.New(logger, sprayLogger), nil
}

func (s state) statusHandler(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path != "/" {
		http.NotFound(w, req)
		return
	}
	writer := bufio.NewWriter(w)
	defer writer.Flush()
	fmt.Fprintln(writer, "<title>installer status page</title>")
	fmt.Fprintln(writer, `<style>
                          table, th, td {
                          border-collapse: collapse;
                          }
                          </style>`)
	fmt.Fprintln(writer, "<body>")
	fmt.Fprintln(writer, "<center>")
	fmt.Fprintln(writer, "<h1>installer status page</h1>")
	fmt.Fprintln(writer, "</center>")
	html.WriteHeaderWithRequest(writer, req)
	fmt.Fprintln(writer, "<h3>")
	s.writeDashboard(writer)
	for _, htmlWriter := range htmlWriters {
		htmlWriter.WriteHtml(writer)
	}
	fmt.Fprintln(writer, "</h3>")
	fmt.Fprintln(writer, "<hr>")
	html.WriteFooter(writer)
	fmt.Fprintln(writer, "</body>")
}

func AddHtmlWriter(htmlWriter HtmlWriter) {
	htmlWriters = append(htmlWriters, htmlWriter)
}

func (s state) writeDashboard(writer io.Writer) {
}

func (t *srpcType) Shell(conn *srpc.Conn) error {
	t.logger.Println("starting shell on SRPC connection")
	t.remoteShellWaitGroup.Add(1)
	defer t.remoteShellWaitGroup.Done()
	pty, tty, err := openPty()
	if err != nil {
		return err
	}
	defer pty.Close()
	defer tty.Close()
	if file, err := os.Open("/var/log/installer/latest"); err != nil {
		t.logger.Println(err)
	} else {
		fmt.Fprintln(conn, "Logs so far:\r")
		// Need to inject carriage returns for each line, so have to do this the
		// hard way.
		reader := bufio.NewReader(file)
		for {
			if chunk, isPrefix, err := reader.ReadLine(); err != nil {
				break
			} else {
				conn.Write(chunk)
				if !isPrefix {
					conn.Write([]byte("\r\n"))
				}
			}
		}
		file.Close()
		conn.Flush()
	}
	cmd := exec.Command("/bin/busybox", "sh", "-i")
	cmd.Env = make([]string, 0)
	cmd.Stdin = tty
	cmd.Stdout = tty
	cmd.Stderr = tty
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	fmt.Fprintln(conn, "Starting shell...\r")
	conn.Flush()
	killed := false
	go func() { // Read from pty until killed.
		for {
			t.mutex.Lock()
			t.connections[conn] = struct{}{}
			t.mutex.Unlock()
			buffer := make([]byte, 256)
			if nRead, err := pty.Read(buffer); err != nil {
				if killed {
					break
				}
				t.logger.Printf("error reading from pty: %s", err)
				break
			} else if _, err := conn.Write(buffer[:nRead]); err != nil {
				t.logger.Printf("error writing to connection: %s\n", err)
				break
			}
			if err := conn.Flush(); err != nil {
				t.logger.Printf("error flushing connection: %s\n", err)
				break
			}
		}
		t.mutex.Lock()
		delete(t.connections, conn)
		t.mutex.Unlock()
	}()
	// Read from connection, write to pty.
	for {
		buffer := make([]byte, 256)
		if nRead, err := conn.Read(buffer); err != nil {
			if err == io.EOF {
				break
			}
			return err
		} else {
			if _, err := pty.Write(buffer[:nRead]); err != nil {
				return err
			}
		}
	}
	killed = true
	cmd.Process.Kill()
	cmd.Wait()
	t.logger.Println("shell for SRPC connection exited")
	return nil
}

func (t *srpcType) Write(p []byte) (int, error) {
	buffer := make([]byte, 0, len(p)+1)
	for _, ch := range p { // First add a carriage return for each newline.
		if ch == '\n' {
			buffer = append(buffer, '\r')
		}
		buffer = append(buffer, ch)
	}
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	for conn := range t.connections {
		conn.Write(buffer)
		conn.Flush()
	}
	return len(p), nil
}
