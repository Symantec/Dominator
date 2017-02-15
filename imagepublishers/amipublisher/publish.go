package amipublisher

import (
	"errors"
	iclient "github.com/Symantec/Dominator/imageserver/client"
	uclient "github.com/Symantec/Dominator/imageunpacker/client"
	"github.com/Symantec/Dominator/lib/awsutil"
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/imageunpacker"
	"github.com/aws/aws-sdk-go/service/ec2"
	"path"
)

func (pData *publishData) publish(targets awsutil.TargetList,
	skipList awsutil.TargetList, logger log.Logger) (
	Results, error) {
	fs, err := pData.getFileSystem(logger)
	if err != nil {
		return nil, err
	}
	fs.TotalDataBytes = fs.EstimateUsage(0)
	pData.fileSystem = fs
	resultsChannel := make(chan TargetResult, 1)
	numTargets, err := awsutil.ForEachTarget(targets, skipList,
		func(awsService *ec2.EC2, account, region string, logger log.Logger) {
			pData.publishToTargetWrapper(awsService, account, region,
				resultsChannel, logger)
		},
		logger)
	// Collect results.
	results := make(Results, 0, numTargets)
	for i := 0; i < numTargets; i++ {
		result := <-resultsChannel
		if result.AccountName == "" || result.Region == "" {
			continue
		}
		results = append(results, result)
	}
	return results, err
}

func (pData *publishData) getFileSystem(logger log.Logger) (
	*filesystem.FileSystem, error) {
	imageName := path.Join(pData.streamName, pData.imageLeafName)
	logger.Printf("Loading image: %s...\n", imageName)
	srpcClient, err := srpc.DialHTTP("tcp", pData.imageServerAddress, 0)
	if err != nil {
		return nil, err
	}
	defer srpcClient.Close()
	image, err := iclient.GetImage(srpcClient, imageName)
	if err != nil {
		return nil, err
	}
	if image == nil {
		return nil, errors.New("image: " + imageName + " not found")
	}
	logger.Printf("Loaded image: %s\n", imageName)
	return image.FileSystem, nil
}

func (pData *publishData) publishToTargetWrapper(awsService *ec2.EC2,
	accountProfileName string, region string, channel chan<- TargetResult,
	logger log.Logger) {
	target := awsutil.Target{AccountName: accountProfileName, Region: region}
	resultMsg := TargetResult{Target: target}
	resultMsg.SnapshotId, resultMsg.AmiId, resultMsg.Size, resultMsg.Error =
		pData.publishToTarget(awsService, logger)
	if resultMsg.Error != nil {
		logger.Println(resultMsg.Error)
	}
	channel <- resultMsg
}

func (pData *publishData) publishToTarget(awsService *ec2.EC2,
	logger log.Logger) (string, string, uint, error) {
	unpackerInstance, srpcClient, err := getWorkingUnpacker(awsService,
		pData.unpackerName, logger)
	if err != nil {
		return "", "", 0, err
	}
	defer srpcClient.Close()
	logger.Printf("Preparing to unpack: %s\n", pData.streamName)
	uclient.PrepareForUnpack(srpcClient, pData.streamName, true, false)
	minBytes := pData.fileSystem.TotalDataBytes + pData.minFreeBytes
	status, err := selectVolume(srpcClient, awsService, pData.streamName,
		minBytes, pData.tags, unpackerInstance, logger)
	if err != nil {
		return "", "", 0, err
	}
	volumeId := status.ImageStreams[pData.streamName].DeviceId
	if status.ImageStreams[pData.streamName].Status !=
		proto.StatusStreamScanned {
		logger.Printf("Preparing to unpack again: %s\n", pData.streamName)
		err := uclient.PrepareForUnpack(srpcClient, pData.streamName, true,
			false)
		if err != nil {
			return "", "", 0, err
		}
	}
	logger.Printf("Unpacking: %s\n", pData.streamName)
	err = uclient.UnpackImage(srpcClient, pData.streamName, pData.imageLeafName)
	if err != nil {
		return "", "", 0, err
	}
	logger.Printf("Capturing: %s\n", pData.streamName)
	err = uclient.PrepareForCapture(srpcClient, pData.streamName)
	if err != nil {
		return "", "", 0, err
	}
	imageName := path.Join(pData.streamName, path.Base(pData.imageLeafName))
	snapshotId, err := createSnapshot(awsService, volumeId, imageName,
		pData.tags, logger)
	if err != nil {
		return "", "", 0, err
	}
	// Kick off scan for next time.
	err = uclient.PrepareForUnpack(srpcClient, pData.streamName, false, true)
	if err != nil {
		return "", "", 0, err
	}
	logger.Println("Registering AMI...")
	volumeSize := status.Devices[volumeId].Size
	amiId, err := registerAmi(awsService, snapshotId, pData.amiName, imageName,
		pData.tags, logger)
	return snapshotId, amiId, uint(volumeSize >> 30), nil
}

func selectVolume(srpcClient *srpc.Client, awsService *ec2.EC2,
	streamName string, minBytes uint64, tags map[string]string,
	instance *ec2.Instance, logger log.Logger) (
	proto.GetStatusResponse, error) {
	status, err := uclient.GetStatus(srpcClient)
	if err != nil {
		return proto.GetStatusResponse{}, err
	}
	// Check if associated device is large enough.
	if streamInfo, ok := status.ImageStreams[streamName]; ok {
		if deviceInfo, ok := status.Devices[streamInfo.DeviceId]; ok {
			if minBytes <= deviceInfo.Size {
				return status, nil
			}
		}
	}
	// Search for an unassociated device which is large enough.
	for deviceId, deviceInfo := range status.Devices {
		if deviceInfo.StreamName == "" && minBytes <= deviceInfo.Size {
			err := uclient.AssociateStreamWithDevice(srpcClient, streamName,
				deviceId)
			if err != nil {
				return proto.GetStatusResponse{}, err
			}
			return uclient.GetStatus(srpcClient)
		}
	}
	// Need to attach another volume.
	volumeId, err := addVolume(srpcClient, awsService, minBytes, tags, instance,
		logger)
	if err != nil {
		return proto.GetStatusResponse{}, err
	}
	err = uclient.AssociateStreamWithDevice(srpcClient, streamName, volumeId)
	if err != nil {
		return proto.GetStatusResponse{}, err
	}
	return uclient.GetStatus(srpcClient)
}

func addVolume(srpcClient *srpc.Client, awsService *ec2.EC2,
	minBytes uint64, tags map[string]string,
	instance *ec2.Instance, logger log.Logger) (string, error) {
	volumeId, err := createVolume(awsService,
		instance.Placement.AvailabilityZone, minBytes, tags, logger)
	if err != nil {
		return "", err
	}
	err = uclient.AddDevice(srpcClient, volumeId, func() error {
		return attachVolume(awsService, instance, volumeId, logger)
	})
	if err != nil {
		return "", err
	}
	return volumeId, nil
}
