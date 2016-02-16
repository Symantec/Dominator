package main

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/imageserver/client"
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/filesystem/scanner"
	"github.com/Symantec/Dominator/lib/filesystem/untar"
	"github.com/Symantec/Dominator/lib/filter"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/image"
	"github.com/Symantec/Dominator/lib/objectclient"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/imageserver"
	"io"
	"net/rpc"
	"os"
	"strings"
)

func addImagefileSubcommand(args []string) {
	imageClient, imageSClient, objectClient := getClients()
	err := addImagefile(imageClient, imageSClient, objectClient, args[0],
		args[1], args[2], args[3])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error adding image: \"%s\"\t%s\n", args[0], err)
		os.Exit(1)
	}
	os.Exit(0)
}

func addImagefile(imageClient *rpc.Client, imageSClient *srpc.Client,
	objectClient *objectclient.ObjectClient,
	name, imageFilename, filterFilename, triggersFilename string) error {
	var request imageserver.AddImageRequest
	var reply imageserver.AddImageResponse
	imageExists, err := checkImage(imageClient, name)
	if err != nil {
		return errors.New("error checking for image existance: " + err.Error())
	}
	if imageExists {
		return errors.New("image exists")
	}
	var newImage image.Image
	if err := loadImageFiles(&newImage, objectClient, filterFilename,
		triggersFilename); err != nil {
		return err
	}
	request.ImageName = name
	request.Image = &newImage
	request.Image.FileSystem, err = buildImage(objectClient, newImage.Filter,
		imageFilename)
	if err != nil {
		return errors.New("error building image: " + err.Error())
	}
	err = client.CallAddImage(imageSClient, request, &reply)
	if err != nil {
		return errors.New("remote error: " + err.Error())
	}
	return nil
}

type hasher struct {
	objQ *objectclient.ObjectAdderQueue
}

func (h *hasher) Hash(reader io.Reader, length uint64) (
	hash.Hash, error) {
	hash, err := h.objQ.Add(reader, length)
	if err != nil {
		return hash, errors.New("error sending image data: " + err.Error())
	}
	return hash, nil
}

func (h *hasher) HandleData(reader io.Reader, length uint64) (
	hash.Hash, error) {
	return h.Hash(reader, length)
}

func buildImage(objectClient *objectclient.ObjectClient, filter *filter.Filter,
	imageFilename string) (*filesystem.FileSystem, error) {
	fi, err := os.Lstat(imageFilename)
	if err != nil {
		return nil, err
	}
	var h hasher
	h.objQ, err = objectclient.NewObjectAdderQueue(objectClient)
	if err != nil {
		return nil, err
	}
	var fs *filesystem.FileSystem
	if fi.IsDir() {
		sfs, err := scanner.ScanFileSystem(imageFilename, nil, filter, nil, &h,
			nil)
		if err != nil {
			h.objQ.Close()
			return nil, err
		}
		fs = &sfs.FileSystem
	} else {
		imageFile, err := os.Open(imageFilename)
		if err != nil {
			h.objQ.Close()
			return nil, errors.New("error opening image file: " + err.Error())
		}
		defer imageFile.Close()
		var imageReader io.Reader
		if strings.HasSuffix(imageFilename, ".tar") {
			imageReader = imageFile
		} else if strings.HasSuffix(imageFilename, ".tar.gz") ||
			strings.HasSuffix(imageFilename, ".tgz") {
			gzipReader, err := gzip.NewReader(imageFile)
			if err != nil {
				h.objQ.Close()
				return nil, errors.New(
					"error creating gzip reader: " + err.Error())
			}
			defer gzipReader.Close()
			imageReader = gzipReader
		} else {
			h.objQ.Close()
			return nil, errors.New("unrecognised image type")
		}
		tarReader := tar.NewReader(imageReader)
		fs, err = untar.Decode(tarReader, &h, filter)
		if err != nil {
			h.objQ.Close()
			return nil, errors.New("error building image: " + err.Error())
		}
	}
	if err != nil {
		h.objQ.Close()
		return nil, err
	}
	err = h.objQ.Close()
	if err != nil {
		return nil, err
	}
	return fs, nil
}
