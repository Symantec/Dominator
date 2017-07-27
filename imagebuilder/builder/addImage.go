package builder

import (
	"bytes"
	"errors"
	"fmt"
	imageclient "github.com/Symantec/Dominator/imageserver/client"
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/filesystem/scanner"
	"github.com/Symantec/Dominator/lib/filesystem/util"
	"github.com/Symantec/Dominator/lib/filter"
	"github.com/Symantec/Dominator/lib/format"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/image"
	"github.com/Symantec/Dominator/lib/log"
	objectclient "github.com/Symantec/Dominator/lib/objectserver/client"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/lib/triggers"
	"io"
	"path"
	"time"
)

const timeFormat = "2006-01-02:15:04:05"

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

func addImage(client *srpc.Client, streamName, dirname string,
	scanFilter *filter.Filter, computedFilesList []util.ComputedFile,
	imageFilter *filter.Filter,
	trig *triggers.Triggers, expiresIn time.Duration,
	buildLog *bytes.Buffer, logger log.Logger) (string, error) {
	buildStartTime := time.Now()
	fs, err := buildFileSystem(client, dirname, scanFilter)
	if err != nil {
		return "", err
	}
	if err := util.SpliceComputedFiles(fs, computedFilesList); err != nil {
		return "", err
	}
	fs.ComputeTotalDataBytes()
	duration := time.Since(buildStartTime)
	speed := uint64(float64(fs.TotalDataBytes) / duration.Seconds())
	fmt.Fprintf(buildLog,
		"Scanned file-system and uploaded %d objects (%s) in %s (%s/s)\n",
		len(fs.InodeTable), format.FormatBytes(fs.TotalDataBytes),
		format.Duration(duration), format.FormatBytes(speed))
	objClient := objectclient.AttachObjectClient(client)
	// Make a copy of the build log because AddObject() drains the buffer.
	buildLog = bytes.NewBuffer(buildLog.Bytes())
	hashVal, _, err := objClient.AddObject(buildLog, uint64(buildLog.Len()),
		nil)
	if err != nil {
		return "", err
	}
	if err := objClient.Close(); err != nil {
		return "", err
	}
	if _, oldImage, err := getLatestImage(client, streamName); err != nil {
		return "", err
	} else if oldImage != nil {
		util.CopyMtimes(oldImage.FileSystem, fs)
	}
	img := &image.Image{
		BuildLog:   &image.Annotation{Object: &hashVal},
		FileSystem: fs,
		Filter:     imageFilter,
		Triggers:   trig,
	}
	if expiresIn > 0 {
		img.ExpiresAt = time.Now().Add(expiresIn)
	}
	if err := img.Verify(); err != nil {
		return "", err
	}
	name := path.Join(streamName, time.Now().Format(timeFormat))
	if err := imageclient.AddImage(client, name, img); err != nil {
		return "", errors.New("remote error: " + err.Error())
	}
	return name, nil
}

func buildFileSystem(client *srpc.Client, dirname string,
	scanFilter *filter.Filter) (
	*filesystem.FileSystem, error) {
	var h hasher
	var err error
	h.objQ, err = objectclient.NewObjectAdderQueue(client)
	if err != nil {
		return nil, err
	}
	fs, err := buildFileSystemWithHasher(dirname, &h, scanFilter)
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

func buildFileSystemWithHasher(dirname string, h *hasher,
	scanFilter *filter.Filter) (
	*filesystem.FileSystem, error) {
	fs, err := scanner.ScanFileSystem(dirname, nil, scanFilter, nil, h, nil)
	if err != nil {
		return nil, err
	}
	return &fs.FileSystem, nil
}
