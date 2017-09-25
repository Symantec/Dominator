package unpacker

import (
	"errors"
	"path"
	"syscall"

	proto "github.com/Symantec/Dominator/proto/imageunpacker"
)

func (u *Unpacker) prepareForCopy(streamName string) error {
	u.updateUsageTime()
	defer u.updateUsageTime()
	streamInfo := u.getStream(streamName)
	if streamInfo == nil {
		return errors.New("unknown stream")
	}
	errorChannel := make(chan error)
	request := requestType{
		request:      requestPrepareForCopy,
		errorChannel: errorChannel,
	}
	streamInfo.requestChannel <- request
	return <-errorChannel
}

func (stream *streamManagerState) prepareForCopy() error {
	if err := stream.getDevice(); err != nil {
		return err
	}
	streamInfo := stream.streamInfo
	switch streamInfo.status {
	case proto.StatusStreamNoDevice:
		return errors.New("no device")
	case proto.StatusStreamNotMounted:
		return nil // Nothing to do.
	case proto.StatusStreamMounted:
		// Unmount.
	case proto.StatusStreamScanning:
		return errors.New("stream scan in progress")
	case proto.StatusStreamScanned:
		// Unmount.
	case proto.StatusStreamFetching:
		return errors.New("fetch in progress")
	case proto.StatusStreamUpdating:
		return errors.New("update in progress")
	case proto.StatusStreamPreparing:
		return errors.New("preparing to capture")
	default:
		panic("invalid status")
	}
	mountPoint := path.Join(stream.unpacker.baseDir, "mnt")
	if err := syscall.Unmount(mountPoint, 0); err != nil {
		return err
	}
	stream.streamInfo.status = proto.StatusStreamNotMounted
	stream.unpacker.logger.Printf("Unmounted(%s)\n", stream.streamName)
	return nil
}
