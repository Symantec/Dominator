package client

import (
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/imageunpacker"
)

func AddDevice(client *srpc.Client, deviceId string, adder func() error) error {
	return addDevice(client, deviceId, adder)
}

func AssociateStreamWithDevice(srpcClient *srpc.Client, streamName string,
	deviceId string) error {
	return associateStreamWithDevice(srpcClient, streamName, deviceId)
}

func GetStatus(srpcClient *srpc.Client) (proto.GetStatusResponse, error) {
	return getStatus(srpcClient)
}

func PrepareForCapture(srpcClient *srpc.Client, streamName string) error {
	return prepareForCapture(srpcClient, streamName)
}

func PrepareForUnpack(srpcClient *srpc.Client, streamName string,
	skipIfPrepared bool, doNotWaitForResult bool) error {
	return prepareForUnpack(srpcClient, streamName, skipIfPrepared,
		doNotWaitForResult)
}

func UnpackImage(srpcClient *srpc.Client, streamName,
	imageLeafName string) error {
	return unpackImage(srpcClient, streamName, imageLeafName)
}
