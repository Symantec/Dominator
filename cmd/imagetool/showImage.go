package main

import (
	"fmt"
	"os"
)

func showImageSubcommand(args []string) {
	if err := showImage(args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "Error showing image\t%s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func showImage(image string) error {
	fs, err := getTypedImage(image)
	if err != nil {
		return err
	}
	return fs.Listf(os.Stdout, listSelector, listFilter)
}
