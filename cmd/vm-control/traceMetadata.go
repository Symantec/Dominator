package main

import (
	"encoding/gob"
	"errors"
	"fmt"
	"net"
	"os"

	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
)

func traceVmMetadataSubcommand(args []string, logger log.DebugLogger) {
	if err := traceVmMetadata(args[0], logger); err != nil {
		fmt.Fprintf(os.Stderr, "Error tracing VM metadata: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func traceVmMetadata(vmHostname string, logger log.DebugLogger) error {
	if vmIP, hypervisor, err := lookupVmAndHypervisor(vmHostname); err != nil {
		return err
	} else {
		return traceVmMetadataOnHypervisor(hypervisor, vmIP, logger)
	}
}

func traceVmMetadataOnHypervisor(hypervisor string, ipAddr net.IP,
	logger log.DebugLogger) error {
	client, err := dialHypervisor(hypervisor)
	if err != nil {
		return err
	}
	defer client.Close()
	return doTraceMetadata(client, ipAddr, logger)
}

func doTraceMetadata(client *srpc.Client, ipAddr net.IP,
	logger log.Logger) error {
	request := proto.TraceVmMetadataRequest{ipAddr}
	conn, err := client.Call("Hypervisor.TraceVmMetadata")
	if err != nil {
		return err
	}
	defer conn.Close()
	encoder := gob.NewEncoder(conn)
	decoder := gob.NewDecoder(conn)
	if err := encoder.Encode(request); err != nil {
		return err
	}
	if err := conn.Flush(); err != nil {
		return err
	}
	var reply proto.TraceVmMetadataResponse
	if err := decoder.Decode(&reply); err != nil {
		return err
	}
	if reply.Error != "" {
		return errors.New(reply.Error)
	}
	for {
		if line, err := conn.ReadString('\n'); err != nil {
			return err
		} else {
			if line == "\n" {
				return nil
			}
			logger.Print(line)
		}
	}
	return nil
}

func maybeWatchVm(client *srpc.Client, hypervisor string, ipAddr net.IP,
	logger log.DebugLogger) error {
	if !*traceMetadata && *probePortNum < 1 {
		return nil
	} else if *traceMetadata && *probePortNum < 1 {
		return doTraceMetadata(client, ipAddr, logger)
	} else if !*traceMetadata && *probePortNum > 0 {
		return probeVmPortOnHypervisorClient(client, ipAddr, logger)
	} else { // Do both.
		go doTraceMetadata(client, ipAddr, logger)
		return probeVmPortOnHypervisor(hypervisor, ipAddr, logger)
	}
}
