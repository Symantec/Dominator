package unpacker

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Cloud-Foundations/Dominator/lib/wsyscall"
	proto "github.com/Cloud-Foundations/Dominator/proto/imageunpacker"
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
		var rootLabel string
		var err error
		if streamInfo.DeviceId != "" {
			rootLabel, err = u.getExt2fsLabel(streamInfo)
			if err == nil && strings.HasPrefix(rootLabel, "rootfs@") {
				streamInfo.status = proto.StatusStreamNotMounted
			} else {
				streamInfo.status = proto.StatusStreamNoFileSystem
				rootLabel = ""
			}
		}
		requestChannel := make(chan requestType)
		streamInfo.requestChannel = requestChannel
		go u.streamManager(streamName, streamInfo, rootLabel, requestChannel)
	}
	return streamInfo, nil
}

// This must be called with the lock held.
func (u *Unpacker) getExt2fsLabel(streamInfo *imageStreamInfo) (string, error) {
	device := u.pState.Devices[streamInfo.DeviceId]
	deviceNode := filepath.Join("/dev", device.DeviceName)
	rootDevice, err := getPartition(deviceNode)
	if err != nil {
		return "", err
	}
	return getExt2fsLabel(rootDevice)
}

func (u *Unpacker) streamManager(streamName string,
	streamInfo *imageStreamInfo, rootLabel string,
	requestChannel <-chan requestType) {
	if err := wsyscall.UnshareMountNamespace(); err != nil {
		panic("Unable to unshare mount namesace: " + err.Error())
	}
	stream := streamManagerState{
		unpacker:   u,
		streamName: streamName,
		streamInfo: streamInfo,
		rootLabel:  rootLabel,
	}
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

func getExt2fsLabel(device string) (string, error) {
	cmd := exec.Command("e2label", device)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("error getting label: %s: %s", err, output)
	} else {
		return strings.TrimSpace(string(output)), nil
	}
}
