package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	uclient "github.com/Cloud-Foundations/Dominator/imageunpacker/client"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
)

func addDeviceSubcommand(client *srpc.Client, args []string) {
	if err := addDevice(client, args[0], args[1], args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error adding device: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
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
