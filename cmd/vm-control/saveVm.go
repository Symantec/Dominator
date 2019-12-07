package main

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/format"
	"github.com/Cloud-Foundations/Dominator/lib/fsutil"
	"github.com/Cloud-Foundations/Dominator/lib/json"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/rsync"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	proto "github.com/Cloud-Foundations/Dominator/proto/hypervisor"
)

type directorySaver struct {
	dirname  string
	filename string // If != "", lock all files to this filename.
}

type writeSeekCloser interface {
	io.Closer
	io.WriteSeeker
}

type vmSaver interface {
	CopyToFile(filename string, reader io.Reader, length uint64) error
	OpenReader(filename string) (io.ReadCloser, uint64, error)
	OpenWriter(filename string) (writeSeekCloser, error)
}

func copyVolumeToVmSaver(saver vmSaver, client *srpc.Client, ipAddr net.IP,
	volIndex uint, size uint64, logger log.DebugLogger) error {
	var filename string
	if volIndex == 0 {
		filename = "root"
	} else {
		filename = fmt.Sprintf("secondary-volume.%d", volIndex-1)
	}
	if reader, initialFileSize, err := saver.OpenReader(filename); err != nil {
		return err
	} else {
		if reader != nil {
			defer reader.Close()
		} else {
			if initialFileSize > size {
				return errors.New("file larger than volume")
			}
		}
		if writer, err := saver.OpenWriter(filename); err != nil {
			return err
		} else {
			err := copyVmVolumeToWriter(writer, reader, initialFileSize,
				client, ipAddr, volIndex, size, logger)
			if err != nil {
				writer.Close()
				return err
			}
			return writer.Close()
		}
	}
}

func copyVmVolumeToWriter(writer io.WriteSeeker, reader io.Reader,
	initialFileSize uint64, client *srpc.Client, ipAddr net.IP, volIndex uint,
	size uint64, logger log.DebugLogger) error {
	request := proto.GetVmVolumeRequest{
		IpAddress:   ipAddr,
		VolumeIndex: volIndex,
	}
	conn, err := client.Call("Hypervisor.GetVmVolume")
	if err != nil {
		return err
	}
	defer conn.Close()
	if err := conn.Encode(request); err != nil {
		return fmt.Errorf("error encoding request: %s", err)
	}
	if err := conn.Flush(); err != nil {
		return err
	}
	var response proto.GetVmVolumeResponse
	if err := conn.Decode(&response); err != nil {
		return err
	}
	if err := errors.New(response.Error); err != nil {
		return err
	}
	startTime := time.Now()
	stats, err := rsync.GetBlocks(conn, conn, conn, reader, writer,
		size, initialFileSize)
	if err != nil {
		return err
	}
	duration := time.Since(startTime)
	speed := uint64(float64(stats.NumRead) / duration.Seconds())
	logger.Debugf(0, "sent %s B, received %s/%s B (%.0f * speedup, %s/s)\n",
		format.FormatBytes(stats.NumWritten), format.FormatBytes(stats.NumRead),
		format.FormatBytes(size),
		float64(size)/float64(stats.NumRead+stats.NumWritten),
		format.FormatBytes(speed))
	return nil
}

func encodeJsonToVmSaver(saver vmSaver, filename string,
	data interface{}) error {
	buffer := &bytes.Buffer{}
	if err := json.WriteWithIndent(buffer, "    ", data); err != nil {
		return err
	}
	return saver.CopyToFile(filename, buffer, uint64(buffer.Len()))
}

func saveVmSubcommand(args []string, logger log.DebugLogger) error {
	if err := saveVm(args[0], args[1], logger); err != nil {
		return fmt.Errorf("Error saving VM: %s", err)
	}
	return nil
}

func saveVm(vmHostname, destination string, logger log.DebugLogger) error {
	if vmIP, hypervisor, err := lookupVmAndHypervisor(vmHostname); err != nil {
		return err
	} else {
		return saveVmOnHypervisor(hypervisor, vmIP, destination, logger)
	}
}

func saveVmOnHypervisor(hypervisor string, ipAddr net.IP, destination string,
	logger log.DebugLogger) error {
	client, err := dialHypervisor(hypervisor)
	if err != nil {
		return err
	}
	defer client.Close()
	vmInfo, err := getVmInfoClient(client, ipAddr)
	if err != nil {
		return err
	}
	u, err := url.Parse(destination)
	if err != nil {
		return err
	}
	var saver vmSaver
	if u.Scheme == "dir" {
		if realSaver, err := newDirectorySaver(u.Path); err != nil {
			return err
		} else {
			saver = realSaver
		}
	} else {
		return fmt.Errorf("unknown scheme: %s", u.Scheme)
	}
	logger.Debugln(0, "saving metadata")
	if err := encodeJsonToVmSaver(saver, "info.json", vmInfo); err != nil {
		return err
	}
	conn, length, err := callGetVmUserData(client, ipAddr)
	if err != nil {
		return err
	}
	if length > 0 {
		logger.Debugln(0, "saving user data")
		err = saver.CopyToFile("user-data.raw", conn, length)
	}
	conn.Close()
	if err != nil {
		return err
	}
	for index, volume := range vmInfo.Volumes {
		err := copyVolumeToVmSaver(saver, client, ipAddr, uint(index),
			volume.Size, logger)
		if err != nil {
			return err
		}
	}
	return nil
}

func newDirectorySaver(dirname string) (*directorySaver, error) {
	if dirname != "" {
		if err := os.MkdirAll(dirname, fsutil.DirPerms); err != nil {
			return nil, err
		}
	}
	return &directorySaver{dirname: dirname}, nil
}

func (saver *directorySaver) CopyToFile(filename string, reader io.Reader,
	length uint64) error {
	file, err := os.OpenFile(saver.Filename(filename), os.O_WRONLY|os.O_CREATE,
		fsutil.PrivateFilePerms)
	if err != nil {
		return err
	}
	if _, err := io.CopyN(file, reader, int64(length)); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}

func (saver *directorySaver) Filename(filename string) string {
	if saver.filename != "" {
		return saver.filename
	}
	return filepath.Join(saver.dirname, filename)
}

func (saver *directorySaver) OpenReader(filename string) (
	io.ReadCloser, uint64, error) {
	file, err := os.OpenFile(saver.Filename(filename), os.O_RDONLY, 0)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, 0, nil
		}
		return nil, 0, err
	} else if fi, err := file.Stat(); err != nil {
		file.Close()
		return nil, 0, err
	} else {
		return file, uint64(fi.Size()), nil
	}
}

func (saver *directorySaver) OpenWriter(filename string) (
	writeSeekCloser, error) {
	return os.OpenFile(saver.Filename(filename),
		os.O_WRONLY|os.O_CREATE, fsutil.PrivateFilePerms)
}
