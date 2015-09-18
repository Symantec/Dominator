package main

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/filesystem/untar"
	"github.com/Symantec/Dominator/lib/filter"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/image"
	"github.com/Symantec/Dominator/lib/objectclient"
	"github.com/Symantec/Dominator/lib/triggers"
	"github.com/Symantec/Dominator/proto/imageserver"
	"io"
	"net/rpc"
	"os"
	"strings"
)

func addImageSubcommand(imageClient *rpc.Client,
	objectClient *objectclient.ObjectClient, args []string) {
	err := addImage(imageClient, objectClient, args[0], args[1], args[2],
		args[3])
	if err != nil {
		fmt.Printf("Error adding image: \"%s\"\t%s\n", args[0], err)
		os.Exit(1)
	}
	os.Exit(0)
}

func addImage(imageClient *rpc.Client, objectClient *objectclient.ObjectClient,
	name, imageFilename, filterFilename, triggersFilename string) error {
	var request imageserver.AddImageRequest
	var reply imageserver.AddImageResponse
	imageFile, err := os.Open(imageFilename)
	if err != nil {
		return errors.New("error opening image file: " + err.Error())
	}
	defer imageFile.Close()
	var imageReader io.Reader
	if strings.HasSuffix(imageFilename, ".tar") {
		imageReader = imageFile
	} else if strings.HasSuffix(imageFilename, ".tar.gz") ||
		strings.HasSuffix(imageFilename, ".tgz") {
		gzipReader, err := gzip.NewReader(imageFile)
		if err != nil {
			return errors.New("error creating gzip reader: " + err.Error())
		}
		defer gzipReader.Close()
		imageReader = gzipReader
	} else {
		return errors.New("unrecognised image type")
	}
	imageExists, err := checkImage(imageClient, name)
	if err != nil {
		return errors.New("error checking for image existance: " + err.Error())
	}
	if imageExists {
		return errors.New("image exists")
	}
	var newImage image.Image
	if err := loadFilter(&newImage, filterFilename); err != nil {
		return err
	}
	if err := loadTriggers(&newImage, triggersFilename); err != nil {
		return err
	}
	request.ImageName = name
	request.Image = &newImage
	tarReader := tar.NewReader(imageReader)
	request.Image.FileSystem, err = buildImage(objectClient, tarReader,
		newImage.Filter)
	if err != nil {
		return errors.New("error building image: " + err.Error())
	}
	err = imageClient.Call("ImageServer.AddImage", request, &reply)
	if err != nil {
		return errors.New("remote error: " + err.Error())
	}
	return nil
}

func loadFilter(image *image.Image, filterFilename string) error {
	filterFile, err := os.Open(filterFilename)
	if err != nil {
		return err
	}
	defer filterFile.Close()
	filterLines, err := readLines(filterFile)
	if err != nil {
		return errors.New("error reading filter: " + err.Error())
	}
	image.Filter, err = filter.NewFilter(filterLines)
	return err
}

func loadTriggers(image *image.Image, triggersFilename string) error {
	triggersFile, err := os.Open(triggersFilename)
	if err != nil {
		return err
	}
	defer triggersFile.Close()
	decoder := json.NewDecoder(triggersFile)
	var trig triggers.Triggers
	err = decoder.Decode(&trig.Triggers)
	if err != nil {
		return errors.New("error decoding triggers " + err.Error())
	}
	image.Triggers = &trig
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

type dataHandler struct {
	objQ *objectclient.ObjectAdderQueue
}

func (dh *dataHandler) HandleData(data []byte) (hash.Hash, error) {
	hash, err := dh.objQ.Add(data)
	if err != nil {
		return hash, errors.New("error sending image data: " + err.Error())
	}
	return hash, nil
}

func buildImage(objectClient *objectclient.ObjectClient, tarReader *tar.Reader,
	filter *filter.Filter) (*filesystem.FileSystem, error) {
	var dh dataHandler
	dh.objQ = objectclient.NewObjectAdderQueue(objectClient, 1024*1024*128)
	fs, err := untar.Decode(tarReader, &dh, filter)
	if err != nil {
		return nil, err
	}
	err = dh.objQ.Flush()
	if err != nil {
		return nil, err
	}
	return fs, nil
}
