package main

import (
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/Cloud-Foundations/Dominator/hypervisor/manager"
	"github.com/Cloud-Foundations/Dominator/lib/fsutil"
	"github.com/Cloud-Foundations/Dominator/lib/log"
)

var shutdownVMsOnNextStop bool

type flusher interface {
	Flush() error
}

func acceptControlConnections(m *manager.Manager, listener net.Listener,
	logger log.DebugLogger) {
	for {
		if conn, err := listener.Accept(); err != nil {
			logger.Println(err)
		} else if err := processControlConnection(conn, m, logger); err != nil {
			logger.Println(err)
		}
	}
}

func configureVMsToStopOnNextStop() {
	sendRequest(connectToControl(), "stop-vms-on-next-stop")
}

func connectToControl() net.Conn {
	sockAddr := filepath.Join(*stateDir, "control")
	if conn, err := net.Dial("unix", sockAddr); err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to: %s: %s\n", sockAddr, err)
		os.Exit(1)
		return nil
	} else {
		return conn
	}
}

func listenForControl(m *manager.Manager, logger log.DebugLogger) error {
	sockAddr := filepath.Join(*stateDir, "control")
	os.Remove(sockAddr)
	if listener, err := net.Listen("unix", sockAddr); err != nil {
		return err
	} else {
		if err := os.Chmod(sockAddr, fsutil.PrivateFilePerms); err != nil {
			return err
		}
		go acceptControlConnections(m, listener, logger)
		return nil
	}
}

func processControlConnection(conn net.Conn, m *manager.Manager,
	logger log.DebugLogger) error {
	defer conn.Close()
	buffer := make([]byte, 256)
	if nRead, err := conn.Read(buffer); err != nil {
		return fmt.Errorf("error reading request: %s\n", err)
	} else if nRead < 1 {
		return fmt.Errorf("read short request: %s\n", err)
	} else {
		request := string(buffer[:nRead])
		if request[nRead-1] != '\n' {
			return fmt.Errorf("request not null-terminated: %s\n", request)
		}
		request = request[:nRead-1]
		switch request {
		case "stop":
			if _, err := fmt.Fprintln(conn, "ok"); err != nil {
				return err
			}
			if shutdownVMsOnNextStop {
				m.ShutdownVMsAndExit()
			} else {
				logger.Println("stopping without shutting down VMs")
				if flusher, ok := logger.(flusher); ok {
					flusher.Flush()
				}
				os.Exit(0)
			}
		case "stop-vms-on-next-stop":
			if _, err := fmt.Fprintln(conn, "ok"); err != nil {
				return err
			}
			shutdownVMsOnNextStop = true
		default:
			if _, err := fmt.Fprintln(conn, "bad request"); err != nil {
				return err
			}
		}
	}
	return nil
}

func requestStop() {
	sendRequest(connectToControl(), "stop")
}

func sendRequest(conn net.Conn, request string) {
	if _, err := fmt.Fprintln(conn, request); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing request: %s\n", err)
		os.Exit(1)
	}
	buffer := make([]byte, 256)
	if nRead, err := conn.Read(buffer); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading response: %s\n", err)
		os.Exit(1)
	} else if nRead < 1 {
		fmt.Fprintf(os.Stderr, "Read short response: %s\n", err)
		os.Exit(1)
	} else {
		response := string(buffer[:nRead])
		if response[nRead-1] != '\n' {
			fmt.Fprintf(os.Stderr, "Response not null-terminated: %s\n",
				response)
			os.Exit(1)
		}
		response = response[:nRead-1]
		if response != "ok" {
			fmt.Fprintf(os.Stderr, "Bad response: %s\n", response)
			os.Exit(1)
		} else {
			conn.Read(buffer) // Wait for EOF.
			conn.Close()
			os.Exit(0)
		}
	}
}
