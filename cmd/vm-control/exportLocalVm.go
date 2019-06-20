package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"

	hyperclient "github.com/Symantec/Dominator/hypervisor/client"
	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
)

type vmExporter interface {
	createVm(hostname string, vmInfo proto.ExportLocalVmInfo) error
	destroyVm(hostname string) error
}

type vmExporterExec struct {
	createCommand  string
	destroyCommand string
}

func exportLocalVmSubcommand(args []string, logger log.DebugLogger) error {
	if err := exportLocalVm(args[0], logger); err != nil {
		return fmt.Errorf("Error exporting VM: %s", err)
	}
	return nil
}

func exportLocalVm(vmHostname string, logger log.DebugLogger) error {
	return vmExport(vmHostname, vmExporterExec{*localVmCreate, *localVmDestroy},
		logger)
}

func (exporter vmExporterExec) createVm(hostname string,
	vmInfo proto.ExportLocalVmInfo) error {
	if exporter.createCommand == "" {
		if err := json.WriteWithIndent(os.Stdout, "    ", vmInfo); err != nil {
			return err
		}
		return errors.New("no command specified: debug mode")
	}
	buffer := &bytes.Buffer{}
	if err := json.WriteWithIndent(buffer, "    ", vmInfo); err != nil {
		return err
	}
	cmd := exec.Command(exporter.createCommand, hostname)
	cmd.Stdin = buffer
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (exporter vmExporterExec) destroyVm(hostname string) error {
	cmd := exec.Command(exporter.destroyCommand, hostname)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func vmExport(vmHostname string, exporter vmExporter,
	logger log.DebugLogger) error {
	vmIpAddr, err := lookupIP(vmHostname)
	if err != nil {
		return err
	}
	rootCookie, err := readRootCookie(logger)
	if err != nil {
		return err
	}
	hypervisor := fmt.Sprintf(":%d", *hypervisorPortNum)
	client, err := srpc.DialHTTP("tcp", hypervisor, 0)
	if err != nil {
		return err
	}
	defer client.Close()
	if err := hyperclient.StopVm(client, vmIpAddr, nil); err != nil {
		return err
	}
	doStart := true
	defer func() {
		if doStart {
			if err := hyperclient.StartVm(client, vmIpAddr, nil); err != nil {
				logger.Println(err)
			}
		}
	}()
	vmInfo, err := hyperclient.ExportLocalVm(client, vmIpAddr, rootCookie)
	if err != nil {
		return err
	}
	if err := exporter.createVm(vmHostname, vmInfo); err != nil {
		return err
	}
	response, err := askForInputChoice("Commit VM "+vmIpAddr.String(),
		[]string{"commit", "defer", "abandon"})
	if err != nil {
		return err
	}
	switch response {
	case "abandon":
		if err := exporter.destroyVm(vmHostname); err != nil {
			return err
		}
		return errorCommitAbandoned
	case "commit":
		if err := hyperclient.DestroyVm(client, vmIpAddr, nil); err != nil {
			return err
		}
		doStart = false
		return nil
	case "defer":
		doStart = false
		return errorCommitDeferred
	}
	return fmt.Errorf("invalid response: %s", response)
}
