package unpacker

import (
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/objectcache"
	"github.com/Symantec/Dominator/lib/wsyscall"
	proto "github.com/Symantec/Dominator/proto/imageunpacker"
)

type streamManagerState struct {
	unpacker    *Unpacker
	streamName  string
	streamInfo  *imageStreamInfo
	mounted     bool
	fileSystem  *filesystem.FileSystem
	objectCache objectcache.ObjectCache
}

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
		select {
		case request := <-requestChannel:
			var err error
			switch request.moveToStatus {
			case proto.StatusStreamScanning:
				err = stream.scan()
			case proto.StatusStreamFetching:
				err = stream.unpack(request.imageName, request.desiredFS)
			case proto.StatusStreamPreparing:
			default:
				panic("cannot move to status: " + request.moveToStatus.String())
			}
			request.errorChannel <- err
		}
	}
}

func (u *Unpacker) getStream(streamName string) *imageStreamInfo {
	u.rwMutex.RLock()
	defer u.rwMutex.RUnlock()
	return u.pState.ImageStreams[streamName]
}
