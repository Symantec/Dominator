package main

import (
	"fmt"

	"github.com/Cloud-Foundations/Dominator/imageunpacker/client"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
)

func exportImageSubcommand(srpcClient *srpc.Client, args []string) error {
	err := client.ExportImage(srpcClient, args[0], args[1], args[2])
	if err != nil {
		return fmt.Errorf("Error exporting image: %s", err)
	}
	return nil
}
