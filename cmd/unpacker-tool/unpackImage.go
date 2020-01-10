package main

import (
	"fmt"

	"github.com/Cloud-Foundations/Dominator/imageunpacker/client"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
)

func unpackImageSubcommand(srpcClient *srpc.Client, args []string) error {
	if err := client.UnpackImage(srpcClient, args[0], args[1]); err != nil {
		return fmt.Errorf("Error unpacking image: %s", err)
	}
	return nil
}
