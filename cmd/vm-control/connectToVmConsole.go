package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"

	"github.com/Cloud-Foundations/Dominator/lib/bufwriter"
	"github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	proto "github.com/Cloud-Foundations/Dominator/proto/hypervisor"
)

func connectToVmConsoleSubcommand(args []string,
	logger log.DebugLogger) error {
	if err := connectToVmConsole(args[0], logger); err != nil {
		return fmt.Errorf("Error connecting to VM console: %s", err)
	}
	return nil
}

func connectToVmConsole(vmHostname string, logger log.DebugLogger) error {
	if vmIP, hypervisor, err := lookupVmAndHypervisor(vmHostname); err != nil {
		return err
	} else {
		return connectToVmConsoleOnHypervisor(hypervisor, vmIP, logger)
	}
}

func connectToVmConsoleOnHypervisor(hypervisor string, ipAddr net.IP,
	logger log.DebugLogger) error {
	client, err := dialHypervisor(hypervisor)
	if err != nil {
		return err
	}
	defer client.Close()
	serverConn, err := client.Call("Hypervisor.ConnectToVmConsole")
	if err != nil {
		return err
	}
	defer serverConn.Close()
	request := proto.ConnectToVmConsoleRequest{
		IpAddress: ipAddr,
	}
	if err := serverConn.Encode(request); err != nil {
		return err
	}
	if err := serverConn.Flush(); err != nil {
		return err
	}
	var response proto.ConnectToVmConsoleResponse
	if err := serverConn.Decode(&response); err != nil {
		return err
	}
	if err := errors.New(response.Error); err != nil {
		return err
	}
	listener, err := net.Listen("tcp", "localhost:")
	if err != nil {
		return err
	}
	defer listener.Close()
	_, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		return err
	}
	if *vncViewer == "" {
		fmt.Fprintf(os.Stderr, "Listening on port %s for VNC connection\n",
			port)
	} else {
		cmd := exec.Command(*vncViewer, "::"+port)
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			logger.Println(err)
		} else {
			fmt.Fprintf(os.Stderr, "Listening on port %s for VNC connection\n",
				port)
		}
	}
	clientConn, err := listener.Accept()
	if err != nil {
		return err
	}
	listener.Close()
	closed := false
	go func() { // Copy from server to client.
		_, err := io.Copy(clientConn, serverConn)
		if err != nil && !closed {
			logger.Fatalln(err)
		}
		os.Exit(0)
	}()
	// Copy from client to server.
	_, err = io.Copy(bufwriter.NewAutoFlushWriter(serverConn), clientConn)
	closed = true
	return err
}
