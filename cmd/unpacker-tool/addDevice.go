package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	uclient "github.com/Cloud-Foundations/Dominator/imageunpacker/client"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
)

func addDeviceSubcommand(args []string, logger log.DebugLogger) error {
	if err := addDevice(getClient(), args[0], args[1], args[2:]); err != nil {
		return fmt.Errorf("Error adding device: %s", err)
	}
	return nil
}

func addDevice(client *srpc.Client, deviceId, command string,
	args []string) error {
	return uclient.AddDevice(client, deviceId,
		func() error { return adder(command, args) })
}

func adder(command string, args []string) error {
	cmd := exec.Command(command, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if err != io.EOF {
			return err
		}
	}
	return nil
}
