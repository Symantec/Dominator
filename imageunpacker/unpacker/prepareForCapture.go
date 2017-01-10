package unpacker

import (
	"errors"
	"flag"
	"fmt"
	"github.com/Symantec/Dominator/lib/format"
	"github.com/Symantec/Dominator/lib/fsutil"
	proto "github.com/Symantec/Dominator/proto/imageunpacker"
	"os"
	"os/exec"
	"path"
	"syscall"
	"time"
)

var (
	makeBootableTool = flag.String("makeBootableTool",
		path.Join(os.Getenv("HOME"), "etc", "make-bootable"),
		"Name of tool to make device bootable")
)

func (u *Unpacker) prepareForCapture(streamName string) error {
	u.updateUsageTime()
	defer u.updateUsageTime()
	streamInfo := u.getStream(streamName)
	if streamInfo == nil {
		return errors.New("unknown stream")
	}
	errorChannel := make(chan error)
	request := requestType{
		request:      requestPrepareForCapture,
		errorChannel: errorChannel,
	}
	streamInfo.requestChannel <- request
	return <-errorChannel
}

func (stream *streamManagerState) prepareForCapture() error {
	if err := stream.getDevice(); err != nil {
		return err
	}
	mountPoint := path.Join(stream.unpacker.baseDir, "mnt")
	if err := stream.mount(mountPoint); err != nil {
		return err
	}
	streamInfo := stream.streamInfo
	switch streamInfo.status {
	case proto.StatusStreamNoDevice:
		return errors.New("no device")
	case proto.StatusStreamNotMounted:
		return errors.New("not mounted")
	case proto.StatusStreamMounted:
		// Start preparing.
	case proto.StatusStreamScanning:
		return errors.New("stream scan in progress")
	case proto.StatusStreamScanned:
		return errors.New("stream not idle")
	case proto.StatusStreamFetching:
		return errors.New("fetch in progress")
	case proto.StatusStreamUpdating:
		return errors.New("update in progress")
	case proto.StatusStreamPreparing:
		return errors.New("already preparing to capture")
	default:
		panic("invalid status")
	}
	streamInfo.status = proto.StatusStreamPreparing
	startTime := time.Now()
	err := stream.capture()
	if err != nil {
		stream.streamInfo.status = proto.StatusStreamMounted
		return err
	}
	stream.streamInfo.status = proto.StatusStreamNotMounted
	stream.unpacker.logger.Printf("Prepared for capture(%s) in %s\n",
		stream.streamName, format.Duration(time.Since(startTime)))
	return nil
}

func (stream *streamManagerState) capture() error {
	stream.unpacker.rwMutex.RLock()
	device := stream.unpacker.pState.Devices[stream.streamInfo.DeviceId]
	stream.unpacker.rwMutex.RUnlock()
	deviceNode := path.Join("/dev", device.DeviceName)
	stream.unpacker.logger.Printf("Preparing for capture(%s) on %s\n",
		stream.streamName, deviceNode)
	// First clean out debris.
	mountPoint := path.Join(stream.unpacker.baseDir, "mnt")
	subdDir := path.Join(mountPoint, ".subd")
	if err := os.RemoveAll(subdDir); err != nil {
		return err
	}
	// Ensure tool is available.
	tool := path.Join(stream.unpacker.baseDir, "mnt", "make-bootable")
	if _, err := os.Stat(tool); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		file, err := os.Open(*makeBootableTool)
		if err != nil {
			return err
		}
		defer file.Close()
		fi, err := file.Stat()
		if err != nil {
			return err
		}
		err = fsutil.CopyToFile(tool, dirPerms, file, uint64(fi.Size()))
		if err != nil {
			return err
		}
	}
	cmd := exec.Command("chroot", mountPoint, "/make-bootable", deviceNode)
	output, err := cmd.CombinedOutput()
	if err != nil {
		stream.unpacker.logger.Println("Error preparing: ", string(output))
		return fmt.Errorf("error preparing: %s: %s", err, output)
	}
	os.Remove(tool)
	if err := syscall.Unmount(mountPoint, 0); err != nil {
		return err
	}
	syscall.Sync()
	return nil
}
