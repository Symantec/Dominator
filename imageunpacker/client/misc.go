package client

import (
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/imageunpacker"
	"path"
)

func associateStreamWithDevice(srpcClient *srpc.Client, streamName string,
	deviceId string) error {
	request := proto.AssociateStreamWithDeviceRequest{
		StreamName: streamName,
		DeviceId:   deviceId,
	}
	var reply proto.AssociateStreamWithDeviceResponse
	return srpcClient.RequestReply("ImageUnpacker.AssociateStreamWithDevice",
		request, &reply)
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

func prepareForUnpack(srpcClient *srpc.Client, streamName string,
	skipIfPrepared bool, doNotWaitForResult bool) error {
	request := proto.PrepareForUnpackRequest{
		DoNotWaitForResult: doNotWaitForResult,
		SkipIfPrepared:     skipIfPrepared,
		StreamName:         streamName,
	}
	var reply proto.PrepareForUnpackResponse
	return srpcClient.RequestReply("ImageUnpacker.PrepareForUnpack", request,
		&reply)
}

func unpackImage(srpcClient *srpc.Client, streamName,
	imageLeafName string) error {
	request := proto.UnpackImageRequest{
		StreamName:    path.Clean(streamName),
		ImageLeafName: path.Clean(imageLeafName),
	}
	var reply proto.UnpackImageResponse
	return srpcClient.RequestReply("ImageUnpacker.UnpackImage", request, &reply)
}
