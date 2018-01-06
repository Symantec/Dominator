package main

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"os"

	fstar "github.com/Symantec/Dominator/lib/filesystem/tar"
	objectclient "github.com/Symantec/Dominator/lib/objectserver/client"
)

func tarImageSubcommand(args []string) {
	_, objectClient := getClients()
	outputFilename := ""
	if len(args) > 1 {
		outputFilename = args[1]
	}
	err := tarImageAndWrite(objectClient, args[0], outputFilename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error taring image: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func tarImageAndWrite(objectClient *objectclient.ObjectClient, imageName,
	outputFilename string) error {
	deleteOutfile := true
	output := io.Writer(os.Stdout)
	if outputFilename != "" {
		var err error
		file, err := os.Create(outputFilename)
		if err != nil {
			return err
		}
		writer := bufio.NewWriter(file)
		output = writer
		defer func() {
			writer.Flush()
			file.Close()
			if deleteOutfile {
				os.Remove(outputFilename)
			}
		}()
	}
	if *compress {
		output = gzip.NewWriter(output)
	}
	tarWriter := tar.NewWriter(output)
	defer tarWriter.Close()
	fs, err := getTypedImage(imageName)
	if err != nil {
		return err
	}
	if err := fstar.Encode(tarWriter, fs, objectClient); err != nil {
		return err
	}
	if err := tarWriter.Flush(); err != nil {
		return err
	}
	deleteOutfile = false
	return nil
}
