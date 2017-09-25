package main

import (
	"fmt"
	"os"

	"github.com/Symantec/Dominator/lib/srpc"
)

func showImageSubcommand(args []string) {
	imageSClient, _ := getClients()
	if err := showImage(imageSClient, args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "Error showing image\t%s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func showImage(client *srpc.Client, image string) error {
	fs, err := getFsOfImage(client, image)
	if err != nil {
		return err
	}
	return fs.Listf(os.Stdout, listSelector, listFilter)
}
