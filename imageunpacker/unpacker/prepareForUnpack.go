package unpacker

import (
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/filesystem/scanner"
	"github.com/Symantec/Dominator/lib/format"
	"github.com/Symantec/Dominator/lib/objectcache"
	"github.com/Symantec/Dominator/lib/wsyscall"
	proto "github.com/Symantec/Dominator/proto/imageunpacker"
	"os"
	"path"
	"time"
)

func (u *Unpacker) prepareForUnpack(streamName string, skipIfPrepared bool,
	doNotWaitForResult bool) error {
	u.rwMutex.Lock()
	streamInfo, err := u.setupStream(streamName)
	u.rwMutex.Unlock()
	if err != nil {
		return err
	}
	errorChannel := make(chan error)
	request := requestType{
		request:        requestScan,
		skipIfPrepared: skipIfPrepared,
		errorChannel:   errorChannel,
	}
	streamInfo.requestChannel <- request
	if doNotWaitForResult {
		go func() {
			<-errorChannel
		}()
		return nil
	}
	return <-errorChannel
}

func (stream *streamManagerState) scan(skipIfPrepared bool) error {
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
		// Start scanning.
	case proto.StatusStreamScanning:
		return errors.New("stream scan in progress")
	case proto.StatusStreamScanned:
		if skipIfPrepared {
			return nil
		}
		// Start scanning.
	case proto.StatusStreamFetching:
		return errors.New("fetch in progress")
	case proto.StatusStreamUpdating:
		return errors.New("update in progress")
	case proto.StatusStreamPreparing:
		return errors.New("preparing to capture")
	default:
		panic("invalid status")
	}
	streamInfo.status = proto.StatusStreamScanning
	stream.unpacker.logger.Printf("Initiating scan(%s)\n", stream.streamName)
	startTime := time.Now()
	var err error
	stream.fileSystem, err = stream.scanFS(mountPoint)
	if err != nil {
		return err
	}
	stream.objectCache, err = objectcache.ScanObjectCache(
		path.Join(stream.unpacker.baseDir, "mnt", ".subd", "objects"))
	if err != nil {
		return err
	}
	streamInfo.status = proto.StatusStreamScanned
	stream.unpacker.logger.Printf("Scanned(%s) in %s\n",
		stream.streamName, format.Duration(time.Since(startTime)))
	return nil
}

func (stream *streamManagerState) scanFS(mountPoint string) (
	*filesystem.FileSystem, error) {
	sfs, err := scanner.ScanFileSystem(mountPoint, nil, nil, nil,
		scanner.GetSimpleHasher(false), nil)
	if err != nil {
		return nil, err
	}
	fs := &sfs.FileSystem
	if err := fs.RebuildInodePointers(); err != nil {
		return nil, err
	}
	fs.BuildEntryMap()
	return fs, nil
}

func (stream *streamManagerState) getDevice() error {
	u := stream.unpacker
	u.rwMutex.Lock()
	defer u.rwMutex.Unlock()
	return stream.getDeviceWithLock()
}

func (stream *streamManagerState) getDeviceWithLock() error {
	u := stream.unpacker
	streamInfo := stream.streamInfo
	if streamInfo.DeviceId != "" {
		return nil
	}
	// Search for unused device.
	for deviceId, deviceInfo := range u.pState.Devices {
		if deviceInfo.StreamName == "" {
			deviceInfo.StreamName = stream.streamName
			u.pState.Devices[deviceId] = deviceInfo
			streamInfo.DeviceId = deviceId
			streamInfo.status = proto.StatusStreamNotMounted
			if err := u.writeStateWithLock(); err != nil {
				return err
			}
			break
		}
	}
	if streamInfo.DeviceId == "" {
		return errors.New("no available device")
	}
	return nil
}

func (stream *streamManagerState) mount(mountPoint string) error {
	streamInfo := stream.streamInfo
	switch streamInfo.status {
	case proto.StatusStreamNoDevice:
		panic("no device")
	case proto.StatusStreamNotMounted:
		// Not mounted: go ahead and mount.
	default:
		// Already mounted.
		return nil
	}
	stream.unpacker.rwMutex.RLock()
	device := stream.unpacker.pState.Devices[stream.streamInfo.DeviceId]
	stream.unpacker.rwMutex.RUnlock()
	deviceNode := path.Join("/dev", device.DeviceName+"1")
	err := wsyscall.Mount(deviceNode, mountPoint, "ext4", 0, "")
	if err != nil {
		return fmt.Errorf("error mounting: %s onto: %s: %s", deviceNode,
			mountPoint, err)
	}
	err = os.MkdirAll(path.Join(mountPoint, ".subd", "objects"), dirPerms)
	if err != nil {
		return err
	}
	streamInfo.status = proto.StatusStreamMounted
	stream.unpacker.logger.Printf("Mounted(%s) %s\n",
		stream.streamName, deviceNode)
	return nil
}
