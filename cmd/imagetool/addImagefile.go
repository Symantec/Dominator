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
	objectclient "github.com/Symantec/Dominator/lib/objectserver/client"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/imageserver"
	"io"
	"os"
	"strings"
)

func addImagefileSubcommand(args []string) {
	imageSClient, objectClient := getClients()
	err := addImagefile(imageSClient, objectClient, args[0], args[1], args[2],
		args[3])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error adding image: \"%s\"\t%s\n", args[0], err)
		os.Exit(1)
	}
	os.Exit(0)
}

func addImagefile(imageSClient *srpc.Client,
	objectClient *objectclient.ObjectClient,
	name, imageFilename, filterFilename, triggersFilename string) error {
	imageExists, err := client.CallCheckImage(imageSClient, name)
	if err != nil {
		return errors.New("error checking for image existance: " + err.Error())
	}
	if imageExists {
		return errors.New("image exists")
	}
	newImage := new(image.Image)
	if err := loadImageFiles(newImage, objectClient, filterFilename,
		triggersFilename); err != nil {
		return err
	}
	newImage.FileSystem, err = buildImage(imageSClient, newImage.Filter,
		imageFilename)
	if err != nil {
		return errors.New("error building image: " + err.Error())
	}
	if err := spliceComputedFiles(newImage.FileSystem); err != nil {
		return err
	}
	return addImage(imageSClient, name, newImage)
}

func addImage(imageSClient *srpc.Client, name string,
	image *image.Image) error {
	var request imageserver.AddImageRequest
	var reply imageserver.AddImageResponse
	request.ImageName = name
	request.Image = image
	if err := image.Verify(); err != nil {
		return err
	}
	if err := image.VerifyRequiredPaths(requiredPaths); err != nil {
		return err
	}
	if err := client.CallAddImage(imageSClient, request, &reply); err != nil {
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

func buildImage(imageSClient *srpc.Client, filter *filter.Filter,
	imageFilename string) (*filesystem.FileSystem, error) {
	fi, err := os.Lstat(imageFilename)
	if err != nil {
		return nil, err
	}
	var h hasher
	h.objQ, err = objectclient.NewObjectAdderQueue(imageSClient)
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
