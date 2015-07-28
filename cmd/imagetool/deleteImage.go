package main

import (
	"fmt"
	_ "github.com/Symantec/Dominator/proto/imageserver"
	"net/rpc"
	"os"
)

func deleteImageSubcommand(client *rpc.Client, args []string) {
	err := deleteImage(client, args[0])
	if err != nil {
		fmt.Printf("Error deleting image\t%s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func deleteImage(client *rpc.Client, name string) error {
	return nil
}
