package main

import (
	"fmt"

	"github.com/Cloud-Foundations/Dominator/lib/format"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
)

func estimateImageUsageSubcommand(args []string, logger log.DebugLogger) error {
	imageSClient, _ := getClients()
	if err := estimateImageUsage(imageSClient, args[0]); err != nil {
		return fmt.Errorf("Error estimating image size: %s\n", err)
	}
	return nil
}

func estimateImageUsage(client *srpc.Client, image string) error {
	fs, err := getFsOfImage(client, image)
	if err != nil {
		return err
	}
	_, err = fmt.Println(format.FormatBytes(fs.EstimateUsage(0)))
	return err
}
