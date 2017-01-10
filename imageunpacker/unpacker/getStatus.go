package unpacker

import (
	proto "github.com/Symantec/Dominator/proto/imageunpacker"
	"time"
)

func (u *Unpacker) getStatus() proto.GetStatusResponse {
	u.rwMutex.RLock()
	defer u.rwMutex.RUnlock()
	devices := make(map[string]proto.DeviceInfo)
	imageStreams := make(map[string]proto.ImageStreamInfo,
		len(u.pState.ImageStreams))
	for deviceId, device := range u.pState.Devices {
		devices[deviceId] = proto.DeviceInfo{
			device.DeviceName, device.size, device.StreamName}
	}
	for name, stream := range u.pState.ImageStreams {
		imageStreams[name] = proto.ImageStreamInfo{
			stream.DeviceId, stream.status}
	}
	return proto.GetStatusResponse{devices, imageStreams,
		time.Since(u.lastUsedTime)}
}
