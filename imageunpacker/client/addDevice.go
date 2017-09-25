package client

import (
	"errors"

	"github.com/Symantec/Dominator/lib/srpc"
)

func addDevice(client *srpc.Client, deviceId string, adder func() error) error {
	conn, err := client.Call("ImageUnpacker.AddDevice")
	if err != nil {
		return err
	}
	defer conn.Close()
	response, err := conn.ReadString('\n')
	if err != nil {
		return err
	}
	response = response[:len(response)-1]
	if response != "" {
		return errors.New(response)
	}
	if err := adder(); err != nil {
		return err
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
