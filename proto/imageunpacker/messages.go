package imageunpacker

import "time"

const (
	StatusStreamNoDevice   = 0
	StatusStreamNotMounted = 1
	StatusStreamMounted    = 2
	StatusStreamScanning   = 3
	StatusStreamScanned    = 4
	StatusStreamFetching   = 5
	StatusStreamUpdating   = 6
	StatusStreamPreparing  = 7
)

type DeviceInfo struct {
	DeviceName string
	Size       uint64
	StreamName string
}

// The AddDevice() RPC is an exclusive transaction following this sequence:
// - Server sends string "\n" if Client should proceed with attaching a device
//   to the server, else it sends a string indicating an error
// - Client sends string containing the DeviceID that was just attached
// - Server sends "\n" if device was found, else an error message.
// - End of transaction. Method completes.

type AssociateStreamWithDeviceRequest struct {
	StreamName string
	DeviceId   string
}

type AssociateStreamWithDeviceResponse struct{}

type GetStatusRequest struct{}

type GetStatusResponse struct {
	Devices           map[string]DeviceInfo      // Key: DeviceId.
	ImageStreams      map[string]ImageStreamInfo // Key: StreamName.
	TimeSinceLastUsed time.Duration
}

type ImageStreamInfo struct {
	DeviceId string
	Status   StreamStatus
}

type PrepareForCaptureRequest struct {
	StreamName string
}

type PrepareForCaptureResponse struct{}

type PrepareForUnpackRequest struct {
	StreamName         string
	SkipIfPrepared     bool
	DoNotWaitForResult bool
}

type PrepareForUnpackResponse struct{}

type StreamStatus uint

func (status StreamStatus) String() string {
	return status.string()
}

type UnpackImageRequest struct {
	StreamName    string
	ImageLeafName string
}

type UnpackImageResponse struct{}
