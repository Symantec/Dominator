package main

import (
	"bufio"
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/proto/imageserver"
	"io"
	"net/rpc"
	"os"
	"strings"
)

func addImageSubcommand(client *rpc.Client, args []string) {
	err := addImage(client, args[0], args[1], args[2])
	if err != nil {
		fmt.Printf("Error adding image: \"%s\"\t%s\n", args[0], err)
		os.Exit(1)
	}
	os.Exit(0)
}

func addImage(client *rpc.Client,
	name, imageFilename, filterFilename string) error {
	var request imageserver.AddImageRequest
	var reply imageserver.AddImageResponse
	if strings.HasSuffix(imageFilename, ".tar") {
		request.CompressionType = imageserver.UNCOMPRESSED
	} else if strings.HasSuffix(imageFilename, ".tar.gz") {
		request.CompressionType = imageserver.GZIP
	} else {
		return errors.New("Unrecognised image type")
	}
	imageFile, err := os.Open(imageFilename)
	if err != nil {
		return err
	}
	defer imageFile.Close()
	filterFile, err := os.Open(filterFilename)
	if err != nil {
		return err
	}
	defer filterFile.Close()
	imageExists, err := checkImage(client, name)
	if err != nil {
		return err
	}
	if imageExists {
		return errors.New("image exists")
	}
	request.ImageName = name
	request.Filter, err = readLines(filterFile)
	if err != nil {
		return err
	}
	request.ImageData, err = newDataStreamer(imageFile)
	if err != nil {
		return err
	}
	err = client.Call("ImageServer.AddImage", request, &reply)
	if err != nil {
		return err
	}
	if !reply.Success {
		return errors.New(reply.ErrorString)
	}
	return nil
}

func readLines(reader io.Reader) ([]string, error) {
	scanner := bufio.NewScanner(reader)
	lines := make([]string, 0)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) < 1 {
			continue
		}
		if line[0] == '#' {
			continue
		}
		lines = append(lines, line)
	}
	if err := scanner.Err(); err != nil {
		return lines, err
	}
	return lines, nil
}

type fileStreamer struct {
	size   uint64
	reader io.Reader
}

func init() {
	var fs fileStreamer
	gob.Register(fs)
}

func (s *fileStreamer) GobEncode() ([]byte, error) {
	return nil, nil
}

func newDataStreamer(file *os.File) (*fileStreamer, error) {
	var streamer fileStreamer
	fi, err := file.Stat()
	if err != nil {
		return nil, err
	}
	streamer.size = uint64(fi.Size())
	streamer.reader = bufio.NewReader(file)
	return &streamer, nil
}

func (s *fileStreamer) Size() uint64 {
	return s.size
}

func (s *fileStreamer) Read(p []byte) (n int, err error) {
	return 0, nil
}

func (s *fileStreamer) Write(p []byte) (n int, err error) {
	panic("fileStreamer.Write() called")
	return 0, nil
}
