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
	"github.com/Symantec/Dominator/lib/mbr"
	objectclient "github.com/Symantec/Dominator/lib/objectserver/client"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/lib/srpc/setupclient"
	"github.com/Symantec/Dominator/lib/wsyscall"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

func addImagefileSubcommand(args []string) {
	imageSClient, objectClient := getClients()
	err := addImagefile(imageSClient, objectClient, args[0], args[1], args[2],
		args[3])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error adding image: \"%s\": %s\n", args[0], err)
		os.Exit(1)
	}
	os.Exit(0)
}

func addImagefile(imageSClient *srpc.Client,
	objectClient *objectclient.ObjectClient,
	name, imageFilename, filterFilename, triggersFilename string) error {
	imageExists, err := client.CheckImage(imageSClient, name)
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

func addImage(imageSClient *srpc.Client, name string, img *image.Image) error {
	if *expiresIn > 0 {
		img.ExpiresAt = time.Now().Add(*expiresIn)
	} else {
		img.ExpiresAt = time.Time{}
	}
	if err := img.Verify(); err != nil {
		return err
	}
	if err := img.VerifyRequiredPaths(requiredPaths); err != nil {
		return err
	}
	if err := client.AddImage(imageSClient, name, img); err != nil {
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
	var h hasher
	var err error
	h.objQ, err = objectclient.NewObjectAdderQueue(imageSClient)
	if err != nil {
		return nil, err
	}
	fs, err := buildImageWithHasher(imageSClient, filter, imageFilename, &h)
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

func buildImageWithHasher(imageSClient *srpc.Client, filter *filter.Filter,
	imageFilename string, h *hasher) (*filesystem.FileSystem, error) {
	fi, err := os.Lstat(imageFilename)
	if err != nil {
		return nil, err
	}
	if fi.IsDir() {
		sfs, err := scanner.ScanFileSystem(imageFilename, nil, filter, nil, h,
			nil)
		if err != nil {
			return nil, err
		}
		return &sfs.FileSystem, nil
	}
	imageFile, err := os.Open(imageFilename)
	if err != nil {
		return nil, errors.New("error opening image file: " + err.Error())
	}
	defer imageFile.Close()
	if partitionTable, err := mbr.Decode(imageFile); err != nil {
		if err != io.EOF {
			return nil, err
		} // Else perhaps a tiny tarfile, definitely not a partition table.
	} else if partitionTable != nil {
		return buildImageFromRaw(imageSClient, filter, imageFile,
			partitionTable, h)
	}
	var imageReader io.Reader
	if strings.HasSuffix(imageFilename, ".tar") {
		imageReader = imageFile
	} else if strings.HasSuffix(imageFilename, ".tar.gz") ||
		strings.HasSuffix(imageFilename, ".tgz") {
		gzipReader, err := gzip.NewReader(imageFile)
		if err != nil {
			return nil, errors.New(
				"error creating gzip reader: " + err.Error())
		}
		defer gzipReader.Close()
		imageReader = gzipReader
	} else {
		return nil, errors.New("unrecognised image type")
	}
	tarReader := tar.NewReader(imageReader)
	fs, err := untar.Decode(tarReader, h, filter)
	if err != nil {
		return nil, errors.New("error building image: " + err.Error())
	}
	return fs, nil
}

func buildImageFromRaw(imageSClient *srpc.Client, filter *filter.Filter,
	imageFile *os.File, partitionTable *mbr.Mbr,
	h *hasher) (*filesystem.FileSystem, error) {
	var index uint
	var offsetOfLargest, sizeOfLargest uint64
	numPartitions := partitionTable.GetNumPartitions()
	for index = 0; index < numPartitions; index++ {
		offset := partitionTable.GetPartitionOffset(index)
		size := partitionTable.GetPartitionSize(index)
		if size > sizeOfLargest {
			offsetOfLargest = offset
			sizeOfLargest = size
		}
	}
	if sizeOfLargest < 1 {
		return nil, errors.New("unable to find largest partition")
	}
	if err := wsyscall.UnshareMountNamespace(); err != nil {
		if os.IsPermission(err) {
			// Try again with sudo(8).
			args := make([]string, 0, len(os.Args)+1)
			if sudoPath, err := exec.LookPath("sudo"); err != nil {
				return nil, err
			} else {
				args = append(args, sudoPath)
			}
			if myPath, err := exec.LookPath(os.Args[0]); err != nil {
				return nil, err
			} else {
				args = append(args, myPath)
			}
			args = append(args, fmt.Sprintf("-certDirectory=%s",
				setupclient.GetCertDirectory()))
			args = append(args, os.Args[1:]...)
			if err := syscall.Exec(args[0], args, os.Environ()); err != nil {
				return nil, errors.New("unable to Exec: " + err.Error())
			}
		}
		return nil, errors.New(
			"error unsharing mount namespace: " + err.Error())
	}
	cmd := exec.Command("mount", "-o",
		fmt.Sprintf("loop,offset=%d", offsetOfLargest), imageFile.Name(),
		"/mnt")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	fs, err := buildImageWithHasher(imageSClient, filter, "/mnt", h)
	syscall.Unmount("/mnt", 0)
	return fs, err
}
