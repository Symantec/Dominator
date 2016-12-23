package amipublisher

import (
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/imageunpacker"
)

func associateStreamWithDevice(srpcClient *srpc.Client, streamName string,
	deviceId string) error {
	request := proto.AssociateStreamWithDeviceRequest{
		StreamName: streamName,
		DeviceId:   deviceId,
	}
	var reply proto.AssociateStreamWithDeviceResponse
	return srpcClient.RequestReply(
		"ImageUnpacker.AssociateStreamWithDevice", request, &reply)
}

func getStatus(srpcClient *srpc.Client) (proto.GetStatusResponse, error) {
	var request proto.GetStatusRequest
	var reply proto.GetStatusResponse
	err := srpcClient.RequestReply("ImageUnpacker.GetStatus", request, &reply)
	return reply, err
}

func prepareForCapture(srpcClient *srpc.Client, streamName string) error {
	request := proto.PrepareForCaptureRequest{StreamName: streamName}
	var reply proto.PrepareForCaptureResponse
	return srpcClient.RequestReply("ImageUnpacker.PrepareForCapture", request,
		&reply)
}

func prepareForUnpack(srpcClient *srpc.Client, streamName string) error {
	request := proto.PrepareForUnpackRequest{
		StreamName:     streamName,
		SkipIfPrepared: true,
	}
	var reply proto.PrepareForUnpackResponse
	return srpcClient.RequestReply("ImageUnpacker.PrepareForUnpack", request,
		&reply)
}

func startScan(srpcClient *srpc.Client, streamName string) error {
	request := proto.PrepareForUnpackRequest{
		StreamName:         streamName,
		DoNotWaitForResult: true,
	}
	var reply proto.PrepareForUnpackResponse
	return srpcClient.RequestReply("ImageUnpacker.PrepareForUnpack", request,
		&reply)
}

func unpack(srpcClient *srpc.Client, streamName, imageLeafName string) error {
	request := proto.UnpackImageRequest{
		StreamName:    streamName,
		ImageLeafName: imageLeafName,
	}
	var reply proto.UnpackImageResponse
	return srpcClient.RequestReply("ImageUnpacker.UnpackImage", request, &reply)
}
