package main

import (
	"fmt"
	"os"

	"github.com/Cloud-Foundations/Dominator/lib/format"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
)

func estimateImageUsageSubcommand(args []string) {
	imageSClient, _ := getClients()
	if err := estimateImageUsage(imageSClient, args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "Error estimating image size: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func estimateImageUsage(client *srpc.Client, image string) error {
	fs, err := getFsOfImage(client, image)
	if err != nil {
		return err
	}
	_, err = fmt.Println(format.FormatBytes(fs.EstimateUsage(0)))
	return err
}
