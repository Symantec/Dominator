package main

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"os"

	"github.com/Symantec/Dominator/imagebuilder/client"
	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/fsutil"
	"github.com/Symantec/Dominator/lib/image"
	"github.com/Symantec/Dominator/lib/log"
	proto "github.com/Symantec/Dominator/proto/imaginator"
)

func buildImageSubcommand(args []string, logger log.DebugLogger) {
	if err := buildImage(args, logger); err != nil {
		fmt.Fprintf(os.Stderr, "Error building image: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func buildImage(args []string, logger log.Logger) error {
	srpcClient := getImaginatorClient()
	request := proto.BuildImageRequest{
		StreamName:     args[0],
		ExpiresIn:      *expiresIn,
		MaxSourceAge:   *maxSourceAge,
		StreamBuildLog: true,
	}
	if len(args) > 1 {
		request.GitBranch = args[1]
	}
	if *imageFilename != "" {
		request.ReturnImage = true
	}
	logBuffer := &bytes.Buffer{}
	var logWriter io.Writer
	if *alwaysShowBuildLog {
		fmt.Fprintln(os.Stderr, "Start of build log ==========================")
		logWriter = os.Stderr
	} else {
		logWriter = logBuffer
	}
	var reply proto.BuildImageResponse
	err := client.BuildImage(srpcClient, request, &reply, logWriter)
	if err != nil {
		if !*alwaysShowBuildLog {
			os.Stderr.Write(logBuffer.Bytes())
		}
		fmt.Fprintln(os.Stderr, "End of build log ============================")
		return err
	}
	if *alwaysShowBuildLog {
		fmt.Fprintln(os.Stderr, "End of build log ============================")
	}
	if *imageFilename != "" {
		if reply.Image == nil {
			if reply.ImageName == "" {
				return errors.New("no image returned: upgrade the Imaginator")
			}
			return fmt.Errorf(
				"image: %s uploaded, not returned: upgrade the Imaginator",
				reply.ImageName)
		}
		return writeImage(reply.Image, *imageFilename)
	}
	fmt.Println(reply.ImageName)
	return nil
}

func writeImage(img *image.Image, filename string) error {
	file, err := fsutil.CreateRenamingWriter(filename, fsutil.PublicFilePerms)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	defer writer.Flush()
	encoder := gob.NewEncoder(writer)
	return encoder.Encode(img)
}
