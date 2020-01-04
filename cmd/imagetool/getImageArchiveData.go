package main

import (
	"bufio"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Cloud-Foundations/Dominator/lib/fsutil"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/imageserver"
)

func getImageArchiveDataSubcommand(args []string,
	logger log.DebugLogger) error {
	imageClient, _ := getClients()
	err := getImageArchiveDataAndWrite(imageClient, args[0], args[1])
	if err != nil {
		return fmt.Errorf("Error getting image: %s\n", err)
	}
	return nil
}

func getImageArchiveDataAndWrite(imageClient *srpc.Client, name,
	outputFilename string) error {
	request := imageserver.GetImageRequest{
		ImageName:        name,
		IgnoreFilesystem: true,
		Timeout:          *timeout,
	}
	var reply imageserver.GetImageResponse
	err := imageClient.RequestReply("ImageServer.GetImage", request, &reply)
	if err != nil {
		return err
	}
	img := reply.Image
	img.Filter = nil
	img.Triggers = nil
	var encoder srpc.Encoder
	if outputFilename == "-" {
		e := json.NewEncoder(os.Stdout)
		e.SetIndent("", "    ")
		encoder = e
	} else {
		file, err := fsutil.CreateRenamingWriter(outputFilename,
			fsutil.PublicFilePerms)
		if err != nil {
			return err
		}
		defer file.Close()
		writer := bufio.NewWriter(file)
		defer writer.Flush()
		if filepath.Ext(outputFilename) == ".json" {
			e := json.NewEncoder(writer)
			e.SetIndent("", "    ")
			encoder = e
		} else {
			encoder = gob.NewEncoder(writer)
		}
	}
	return encoder.Encode(img)
}
