package main

import (
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/lib/srpc"
	"io"
	"os"
	"os/exec"
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
	conn, err := client.Call("ImageUnpacker.AddDevice")
	if err != nil {
		return err
	}
	response, err := conn.ReadString('\n')
	if err != nil {
		return err
	}
	response = response[:len(response)-1]
	if response != "" {
		return errors.New(response)
	}
	cmd := exec.Command(command, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if err != io.EOF {
			return err
		}
	}
	if _, err := conn.WriteString(deviceId + "\n"); err != nil {
		return err
	}
	if err := conn.Flush(); err != nil {
		return err
	}
	response, err = conn.ReadString('\n')
	if err != nil {
		return err
	}
	response = response[:len(response)-1]
	if response != "" {
		return errors.New(response)
	}
	return nil
}
