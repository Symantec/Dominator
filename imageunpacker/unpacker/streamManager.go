package unpacker

import (
	"strconv"

	"github.com/Symantec/Dominator/lib/wsyscall"
	proto "github.com/Symantec/Dominator/proto/imageunpacker"
)

// This must be called with the lock held.
func (u *Unpacker) setupStream(streamName string) (*imageStreamInfo, error) {
	streamInfo := u.pState.ImageStreams[streamName]
	if streamInfo == nil {
		streamInfo = &imageStreamInfo{}
		u.pState.ImageStreams[streamName] = streamInfo
		if err := u.writeStateWithLock(); err != nil {
			return nil, err
		}
	}
	if streamInfo.requestChannel == nil {
		if streamInfo.DeviceId != "" {
			streamInfo.status = proto.StatusStreamNotMounted
		}
		requestChannel := make(chan requestType)
		streamInfo.requestChannel = requestChannel
		go u.streamManager(streamName, streamInfo, requestChannel)
	}
	return streamInfo, nil
}

func (u *Unpacker) streamManager(streamName string,
	streamInfo *imageStreamInfo,
	requestChannel <-chan requestType) {
	if err := wsyscall.UnshareMountNamespace(); err != nil {
		panic("Unable to unshare mount namesace: " + err.Error())
	}
	stream := streamManagerState{
		unpacker:   u,
		streamName: streamName,
		streamInfo: streamInfo}
	for {
		u.rwMutex.Lock()
		streamInfo.scannedFS = stream.fileSystem
		u.rwMutex.Unlock()
		select {
		case request := <-requestChannel:
			var err error
			switch request.request {
			case requestAssociateWithDevice:
				err = stream.associateWithDevice(request.deviceId)
			case requestScan:
				err = stream.scan(request.skipIfPrepared)
			case requestUnpack:
				err = stream.unpack(request.imageName, request.desiredFS)
			case requestPrepareForCapture:
				err = stream.prepareForCapture()
			case requestPrepareForCopy:
				err = stream.prepareForCopy()
			case requestExport:
				err = stream.export(request.exportType,
					request.exportDestination)
			default:
				panic("unknown request: " + strconv.Itoa(request.request))
			}
			request.errorChannel <- err
			if err != nil {
				u.logger.Println(err)
			}
		}
	}
}

func (u *Unpacker) getStream(streamName string) *imageStreamInfo {
	u.rwMutex.RLock()
	defer u.rwMutex.RUnlock()
	return u.pState.ImageStreams[streamName]
}
