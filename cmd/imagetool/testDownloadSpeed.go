package main

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"time"

	imgclient "github.com/Cloud-Foundations/Dominator/imageserver/client"
	"github.com/Cloud-Foundations/Dominator/lib/filesystem"
	"github.com/Cloud-Foundations/Dominator/lib/format"
	"github.com/Cloud-Foundations/Dominator/lib/hash"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/objectserver"
)

func testDownloadSpeedSubcommand(args []string, logger log.DebugLogger) error {
	if err := testDownloadSpeed(args[0], logger); err != nil {
		return fmt.Errorf("Error testing download speed: %s\n", err)
	}
	return nil
}

func testDownloadSpeed(imageName string, logger log.Logger) error {
	imageSClient, objClient := getClients()
	isDir, err := imgclient.CheckDirectory(imageSClient, imageName)
	if err != nil {
		return err
	}
	if isDir {
		name, err := imgclient.FindLatestImage(imageSClient, imageName, false)
		if err != nil {
			return err
		} else {
			imageName = name
		}
	}
	startTime := time.Now()
	img, err := imgclient.GetImageWithTimeout(imageSClient, imageName, *timeout)
	if err != nil {
		return err
	}
	if img == nil {
		return errors.New(imageName + ": not found")
	}
	finishedTime := time.Now()
	logger.Printf("downloaded image metadata in %s\n",
		format.Duration(finishedTime.Sub(startTime)))
	hashes := make([]hash.Hash, 0, len(img.FileSystem.InodeTable))
	for _, inode := range img.FileSystem.InodeTable {
		if inode, ok := inode.(*filesystem.RegularInode); ok {
			if inode.Size > 0 {
				hashes = append(hashes, inode.Hash)
			}
		}
	}
	startTime = time.Now()
	objectsReader, err := objClient.GetObjects(hashes)
	if err != nil {
		return err
	}
	defer objectsReader.Close()
	var totalBytes uint64
	buffer := make([]byte, 32<<10)
	for range hashes {
		if length, err := readOneObject(objectsReader, buffer); err != nil {
			return err
		} else {
			totalBytes += length
		}
	}
	finishedTime = time.Now()
	downloadTime := finishedTime.Sub(startTime)
	downloadSpeed := float64(totalBytes) / downloadTime.Seconds()
	logger.Printf("downloaded %s (%d objects) in %s (%s/s)\n",
		format.FormatBytes(totalBytes), len(hashes),
		format.Duration(downloadTime),
		format.FormatBytes(uint64(downloadSpeed)))
	return nil
}

func readOneObject(objectsReader objectserver.ObjectsReader,
	buffer []byte) (uint64, error) {
	_, reader, err := objectsReader.NextObject()
	if err != nil {
		return 0, err
	}
	defer reader.Close()
	nCopied, err := io.CopyBuffer(ioutil.Discard, reader, buffer)
	return uint64(nCopied), err
}
