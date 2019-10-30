package client

import (
	"path"

	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	proto "github.com/Cloud-Foundations/Dominator/proto/imageunpacker"
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

func exportImage(srpcClient *srpc.Client, streamName,
	exportType, exportDestination string) error {
	request := proto.ExportImageRequest{
		StreamName:  path.Clean(streamName),
		Type:        exportType,
		Destination: exportDestination,
	}
	var reply proto.ExportImageResponse
	return srpcClient.RequestReply("ImageUnpacker.ExportImage", request, &reply)
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

func prepareForCopy(srpcClient *srpc.Client, streamName string) error {
	request := proto.PrepareForCopyRequest{StreamName: streamName}
	var reply proto.PrepareForCopyResponse
	return srpcClient.RequestReply("ImageUnpacker.PrepareForCopy", request,
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

func removeDevice(srpcClient *srpc.Client, deviceId string) error {
	request := proto.RemoveDeviceRequest{DeviceId: deviceId}
	var reply proto.RemoveDeviceResponse
	return srpcClient.RequestReply("ImageUnpacker.RemoveDevice", request,
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
