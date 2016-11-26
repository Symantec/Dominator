package unpacker

import (
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/imageunpacker"
	"io"
	"log"
	"sync"
)

var (
	stateFile = "state.json"
)

type deviceInfo struct {
	DeviceName string
	size       uint64
	StreamName string
}

type requestType struct {
	moveToStatus proto.StreamStatus
	errorChannel chan<- error
}

type imageStreamInfo struct {
	DeviceId       string
	status         proto.StreamStatus
	requestChannel chan<- requestType
}

type persistentState struct {
	Devices      map[string]deviceInfo       // Key: DeviceId.
	ImageStreams map[string]*imageStreamInfo // Key: StreamName.
}

type Unpacker struct {
	baseDir             string
	imageServerResource *srpc.ClientResource
	logger              *log.Logger
	rwMutex             sync.RWMutex // Protect below.
	pState              persistentState
	scannedDevices      map[string]struct{}
}

func Load(baseDir string, imageServerAddress string, logger *log.Logger) (
	*Unpacker, error) {
	return load(baseDir, imageServerAddress, logger)
}

func (u *Unpacker) AddDevice(deviceId string) error {
	return u.addDevice(deviceId)
}

func (u *Unpacker) GetStatus() proto.GetStatusResponse {
	return u.getStatus()
}

func (u *Unpacker) PrepareForCapture(streamName string) error {
	return u.prepareForCapture(streamName)
}

func (u *Unpacker) PrepareForUnpack(streamName string) error {
	return u.prepareForUnpack(streamName)
}

func (u *Unpacker) PrepareForAddDevice() error {
	return u.prepareForAddDevice()
}

func (u *Unpacker) WriteHtml(writer io.Writer) {
	u.writeHtml(writer)
}
