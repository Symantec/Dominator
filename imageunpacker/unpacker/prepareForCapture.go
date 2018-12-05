package unpacker

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/filesystem/util"
	"github.com/Symantec/Dominator/lib/format"
	proto "github.com/Symantec/Dominator/proto/imageunpacker"
)

func scanBootDirectory(rootDir string) (*filesystem.FileSystem, error) {
	file, err := os.Open(filepath.Join(rootDir, "boot"))
	if err != nil {
		return nil, err
	}
	defer file.Close()
	names, err := file.Readdirnames(-1)
	if err != nil {
		return nil, err
	}
	bootInode := &filesystem.DirectoryInode{}
	for _, name := range names {
		bootInode.EntryList = append(bootInode.EntryList,
			&filesystem.DirectoryEntry{Name: name})
	}
	bootEntry := &filesystem.DirectoryEntry{Name: "boot"}
	bootEntry.SetInode(bootInode)
	fs := &filesystem.FileSystem{
		DirectoryInode: filesystem.DirectoryInode{
			EntryList: []*filesystem.DirectoryEntry{bootEntry},
		},
	}
	return fs, nil
}

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
	mountPoint := filepath.Join(stream.unpacker.baseDir, "mnt")
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
	deviceNode := filepath.Join("/dev", device.DeviceName)
	stream.unpacker.logger.Printf(
		"Preparing for capture(%s) on %s with label: %s\n",
		stream.streamName, deviceNode, stream.rootLabel)
	// First clean out debris.
	mountPoint := filepath.Join(stream.unpacker.baseDir, "mnt")
	subdDir := filepath.Join(mountPoint, ".subd")
	if err := os.RemoveAll(subdDir); err != nil {
		return err
	}
	fs, err := scanBootDirectory(mountPoint)
	if err != nil {
		stream.unpacker.logger.Printf("Error scanning boot directory: %s\n",
			err)
		return fmt.Errorf("error getting scanning boot directory: %s", err)
	}
	err = util.MakeBootable(fs, deviceNode, stream.rootLabel, mountPoint,
		"net.ifnames=0", false, stream.unpacker.logger)
	if err != nil {
		stream.unpacker.logger.Printf("Error preparing: %s", err)
		return fmt.Errorf("error preparing: %s", err)
	}
	if err := syscall.Unmount(mountPoint, 0); err != nil {
		return err
	}
	syscall.Sync()
	return nil
}
