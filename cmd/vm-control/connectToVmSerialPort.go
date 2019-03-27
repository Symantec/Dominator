package main

import (
	"encoding/gob"
	"fmt"
	"net"
	"os"

	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/log"
	terminalclient "github.com/Symantec/Dominator/lib/net/terminal/client"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
)

func connectToVmSerialPortSubcommand(args []string, logger log.DebugLogger) {
	if err := connectToVmSerialPort(args[0], logger); err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to VM serial port: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func connectToVmSerialPort(vmHostname string, logger log.DebugLogger) error {
	if vmIP, hypervisor, err := lookupVmAndHypervisor(vmHostname); err != nil {
		return err
	} else {
		return connectToVmSerialPortOnHypervisor(hypervisor, vmIP, logger)
	}
}

func connectToVmSerialPortOnHypervisor(hypervisor string, ipAddr net.IP,
	logger log.DebugLogger) error {
	client, err := dialHypervisor(hypervisor)
	if err != nil {
		return err
	}
	defer client.Close()
	conn, err := client.Call("Hypervisor.ConnectToVmSerialPort")
	if err != nil {
		return err
	}
	defer conn.Close()
	encoder := gob.NewEncoder(conn)
	decoder := gob.NewDecoder(conn)
	request := proto.ConnectToVmSerialPortRequest{
		IpAddress:  ipAddr,
		PortNumber: *serialPort,
	}
	if err := encoder.Encode(request); err != nil {
		return err
	}
	if err := conn.Flush(); err != nil {
		return err
	}
	var response proto.ChangeVmTagsResponse
	if err := decoder.Decode(&response); err != nil {
		return err
	}
	if err := errors.New(response.Error); err != nil {
		return err
	}
	if err := terminalclient.StartTerminal(conn); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Fprint(os.Stderr, "\r")
	return nil
}
