package unpacker

import (
	"errors"
	proto "github.com/Symantec/Dominator/proto/imageunpacker"
	"path"
	"syscall"
)

func (u *Unpacker) associateStreamWithDevice(streamName string,
	deviceId string) error {
	defer u.updateUsageTime()
	u.rwMutex.Lock()
	u.updateUsageTimeWithLock()
	streamInfo, err := u.setupStream(streamName)
	u.rwMutex.Unlock()
	if err != nil {
		return err
	}
	errorChannel := make(chan error)
	request := requestType{
		request:      requestAssociateWithDevice,
		deviceId:     deviceId,
		errorChannel: errorChannel,
	}
	streamInfo.requestChannel <- request
	return <-errorChannel
}

func (stream *streamManagerState) associateWithDevice(deviceId string) error {
	streamInfo := stream.streamInfo
	switch streamInfo.status {
	case proto.StatusStreamNoDevice:
		// OK to associate.
	case proto.StatusStreamNotMounted:
		// OK to (re)associate.
	case proto.StatusStreamMounted:
		// OK to (re)associate.
	case proto.StatusStreamScanning:
		return errors.New("stream scan in progress")
	case proto.StatusStreamScanned:
		// OK to (re)associate.
	case proto.StatusStreamFetching:
		return errors.New("fetch in progress")
	case proto.StatusStreamUpdating:
		return errors.New("update in progress")
	case proto.StatusStreamPreparing:
		return errors.New("preparing to capture")
	default:
		panic("invalid status")
	}
	return stream.selectDevice(deviceId)
}

func (stream *streamManagerState) selectDevice(deviceId string) error {
	streamInfo := stream.streamInfo
	u := stream.unpacker
	u.rwMutex.Lock()
	defer u.rwMutex.Unlock()
	if streamInfo.DeviceId == deviceId {
		return nil
	}
	switch streamInfo.status {
	case proto.StatusStreamNoDevice:
		// Nothing to unmount.
	case proto.StatusStreamNotMounted:
		// Not mounted.
	default:
		// Mounted: unmount it.
		mountPoint := path.Join(stream.unpacker.baseDir, "mnt")
		if err := syscall.Unmount(mountPoint, 0); err != nil {
			return err
		}
		streamInfo.status = proto.StatusStreamNotMounted
	}
	if deviceId == "" {
		return stream.getDeviceWithLock()
	}
	if device, ok := u.pState.Devices[deviceId]; !ok {
		return errors.New("unknown device ID: " + deviceId)
	} else {
		if device.StreamName != "" {
			return errors.New(
				"device ID: " + deviceId + " used by: " + device.StreamName)
		}
		if streamInfo.DeviceId != "" { // Disassociate with existing device.
			if device, ok := u.pState.Devices[streamInfo.DeviceId]; ok {
				device.StreamName = ""
				u.pState.Devices[streamInfo.DeviceId] = device
			}
		}
		device.StreamName = stream.streamName
		u.pState.Devices[deviceId] = device
		streamInfo.DeviceId = deviceId
		streamInfo.status = proto.StatusStreamNotMounted
		return u.writeStateWithLock()
	}
}
