package main

import (
	"fmt"

	"github.com/Cloud-Foundations/Dominator/imageunpacker/client"
	"github.com/Cloud-Foundations/Dominator/lib/log"
)

func exportImageSubcommand(args []string, logger log.DebugLogger) error {
	err := client.ExportImage(getClient(), args[0], args[1], args[2])
	if err != nil {
		return fmt.Errorf("Error exporting image: %s", err)
	}
	return nil
}
