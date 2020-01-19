package main

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"os"

	"github.com/Cloud-Foundations/Dominator/lib/filesystem/tar"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	objectclient "github.com/Cloud-Foundations/Dominator/lib/objectserver/client"
)

func tarImageSubcommand(args []string, logger log.DebugLogger) error {
	_, objectClient := getClients()
	outputFilename := ""
	if len(args) > 1 {
		outputFilename = args[1]
	}
	err := tarImageAndWrite(objectClient, args[0], outputFilename)
	if err != nil {
		return fmt.Errorf("Error taring image: %s", err)
	}
	return nil
}

func tarImageAndWrite(objectClient *objectclient.ObjectClient, imageName,
	outputFilename string) error {
	fs, objectsGetter, err := getImageForUnpack(objectClient, imageName)
	if err != nil {
		return err
	}
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
		zWriter := gzip.NewWriter(output)
		defer zWriter.Close()
		output = zWriter
	}
	if err := tar.Write(output, fs, objectsGetter); err != nil {
		return err
	}
	deleteOutfile = false
	return nil
}
