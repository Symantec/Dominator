package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"syscall"

	"github.com/Symantec/Dominator/lib/html"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
)

type HtmlWriter interface {
	WriteHtml(writer io.Writer)
}

type state struct {
	logger log.DebugLogger
}

type srpcType struct {
	logger log.DebugLogger
}

var htmlWriters []HtmlWriter

func startServer(portNum uint, logger log.DebugLogger) error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", portNum))
	if err != nil {
		return err
	}
	myState := state{logger}
	html.HandleFunc("/", myState.statusHandler)
	srpcObj := &srpcType{logger: logger}
	if err := srpc.RegisterName("Installer", srpcObj); err != nil {
		logger.Printf("error registering SRPC receiver: %s\n", err)
	}
	go http.Serve(listener, nil)
	return nil
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
	pty, tty, err := openPty()
	if err != nil {
		return err
	}
	defer pty.Close()
	defer tty.Close()
	cmd := exec.Command("/bin/busybox", "sh", "-i")
	cmd.Env = make([]string, 0)
	cmd.Stdin = tty
	cmd.Stdout = tty
	cmd.Stderr = tty
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	killed := false
	go func() { // Read from pty until killed.
		for {
			buffer := make([]byte, 256)
			if nRead, err := pty.Read(buffer); err != nil {
				if killed {
					return
				}
				t.logger.Printf("error reading from pty: %s", err)
				return
			} else if _, err := conn.Write(buffer[:nRead]); err != nil {
				t.logger.Printf("error writing to connection: %s\n", err)
				return
			}
			if err := conn.Flush(); err != nil {
				t.logger.Printf("error flushing connection: %s\n", err)
				return
			}
		}
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
		if err := conn.Flush(); err != nil {
			return err
		}
	}
	killed = true
	cmd.Process.Kill()
	cmd.Wait()
	t.logger.Println("shell for SRPC connection exited")
	return nil
}
