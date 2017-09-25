package unpacker

import (
	"io"
	"sync"
	"time"

	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/objectcache"
	proto "github.com/Symantec/Dominator/proto/imageunpacker"
)

const (
	requestAssociateWithDevice = iota
	requestScan
	requestUnpack
	requestPrepareForCapture
	requestPrepareForCopy
	requestExport
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
	request           int
	desiredFS         *filesystem.FileSystem
	imageName         string
	deviceId          string
	skipIfPrepared    bool
	exportType        string
	exportDestination string
	errorChannel      chan<- error
}

type imageStreamInfo struct {
	DeviceId       string
	status         proto.StreamStatus
	requestChannel chan<- requestType
	scannedFS      *filesystem.FileSystem
}

type persistentState struct {
	Devices      map[string]deviceInfo       // Key: DeviceId.
	ImageStreams map[string]*imageStreamInfo // Key: StreamName.
}

type streamManagerState struct {
	unpacker    *Unpacker
	streamName  string
	streamInfo  *imageStreamInfo
	fileSystem  *filesystem.FileSystem
	objectCache objectcache.ObjectCache
}

type Unpacker struct {
	baseDir            string
	imageServerAddress string
	logger             log.Logger
	rwMutex            sync.RWMutex // Protect below.
	pState             persistentState
	scannedDevices     map[string]struct{}
	lastUsedTime       time.Time
}

func Load(baseDir string, imageServerAddress string, logger log.Logger) (
	*Unpacker, error) {
	return load(baseDir, imageServerAddress, logger)
}

func (u *Unpacker) AddDevice(deviceId string) error {
	return u.addDevice(deviceId)
}

func (u *Unpacker) AssociateStreamWithDevice(streamName string,
	deviceId string) error {
	return u.associateStreamWithDevice(streamName, deviceId)
}

func (u *Unpacker) ExportImage(streamName string, exportType string,
	exportDestination string) error {
	return u.exportImage(streamName, exportType, exportDestination)
}

func (u *Unpacker) GetFileSystem(streamName string) (
	*filesystem.FileSystem, error) {
	return u.getFileSystem(streamName)
}

func (u *Unpacker) GetStatus() proto.GetStatusResponse {
	return u.getStatus()
}

func (u *Unpacker) PrepareForCapture(streamName string) error {
	return u.prepareForCapture(streamName)
}

func (u *Unpacker) PrepareForCopy(streamName string) error {
	return u.prepareForCopy(streamName)
}

func (u *Unpacker) PrepareForUnpack(streamName string, skipIfPrepared bool,
	doNotWaitForResult bool) error {
	return u.prepareForUnpack(streamName, skipIfPrepared, doNotWaitForResult)
}

func (u *Unpacker) PrepareForAddDevice() error {
	return u.prepareForAddDevice()
}

func (u *Unpacker) RemoveDevice(deviceId string) error {
	return u.removeDevice(deviceId)
}

func (u *Unpacker) UnpackImage(streamName string, imageLeafName string) error {
	return u.unpackImage(streamName, imageLeafName)
}

func (u *Unpacker) WriteHtml(writer io.Writer) {
	u.writeHtml(writer)
}
