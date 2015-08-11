package main

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"crypto/sha512"
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/image"
	"github.com/Symantec/Dominator/proto/imageserver"
	"github.com/Symantec/Dominator/proto/objectserver"
	"io"
	"io/ioutil"
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
	filterFile, err := os.Open(filterFilename)
	if err != nil {
		return err
	}
	defer filterFile.Close()
	imageExists, err := checkImage(client, name)
	if err != nil {
		return errors.New("error checking for image existance: " + err.Error())
	}
	if imageExists {
		return errors.New("image exists")
	}
	var newImage image.Image
	var filter image.Filter
	newImage.Filter = &filter
	request.ImageName = name
	request.Image = &newImage
	filter.FilterLines, err = readLines(filterFile)
	if err != nil {
		return errors.New("error reading filter: " + err.Error())
	}
	tarReader := tar.NewReader(imageReader)
	request.Image.FileSystem, err = buildImage(client, tarReader)
	if err != nil {
		return errors.New("error building image: " + err.Error())
	}
	err = client.Call("ImageServer.AddImage", request, &reply)
	if err != nil {
		return errors.New("remote error: " + err.Error())
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

func buildImage(client *rpc.Client, tarReader *tar.Reader) (
	*filesystem.FileSystem, error) {
	var fs filesystem.FileSystem
	var objQ objectQueue
	objQ.maxBytes = 1024 * 1024 * 128
	objQ.client = client
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if header.Typeflag == tar.TypeReg || header.Typeflag == tar.TypeRegA {
			err := objQ.Add(uint64(header.Size), tarReader)
			if err != nil {
				return nil, errors.New("error sending image data for: " +
					header.Name + ": " + err.Error())
			}
		}
	}
	err := objQ.Flush()
	if err != nil {
		return nil, err
	}
	// TODO(rgooch): Decode image, Call AddFiles() RPC in batches, finally call
	//               AddImage() RPC.
	return &fs, nil
}

type objectQueue struct {
	numBytes uint64
	maxBytes uint64
	client   *rpc.Client
	objects  []*objectserver.AddObjectSubrequest
}

func (objQ *objectQueue) Add(size uint64, reader io.Reader) error {
	if size+objQ.numBytes > objQ.maxBytes {
		err := objQ.Flush()
		if err != nil {
			return err
		}
	}
	var hash hash.Hash
	var object objectserver.AddObjectSubrequest
	object.ExpectedHash = &hash
	var err error
	object.ObjectData, err = ioutil.ReadAll(reader)
	if err != nil {
		return errors.New("error reading file data" + err.Error())
	}
	if uint64(len(object.ObjectData)) != size {
		return errors.New(fmt.Sprintf(
			"failed to read file data, wanted: %d, got: %d bytes", size,
			len(object.ObjectData)))
	}
	hasher := sha512.New()
	_, err = hasher.Write(object.ObjectData)
	if err != nil {
		return err
	}
	copy(hash[:], hasher.Sum(nil))
	objQ.objects = append(objQ.objects, &object)
	objQ.numBytes += size
	return nil
}

func (objQ *objectQueue) Flush() error {
	var request objectserver.AddFilesRequest
	var reply objectserver.AddFilesResponse
	request.ObjectsToAdd = objQ.objects
	// TODO(rgooch): Remove debugging output.
	fmt.Printf("Flushing: %d objects\n", len(request.ObjectsToAdd))
	err := objQ.client.Call("ObjectServer.AddFiles", request, &reply)
	if err != nil {
		return errors.New("error adding files, remote error: " + err.Error())
	}
	objQ.numBytes = 0
	objQ.objects = nil
	return nil
}
