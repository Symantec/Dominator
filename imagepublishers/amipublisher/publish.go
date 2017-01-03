package amipublisher

import (
	"errors"
	iclient "github.com/Symantec/Dominator/imageserver/client"
	uclient "github.com/Symantec/Dominator/imageunpacker/client"
	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/imageunpacker"
	"github.com/aws/aws-sdk-go/service/ec2"
	"path"
	"strconv"
	"time"
)

func (pData *publishData) publish(accountProfileNames []string,
	regions []string, logger log.Logger) (Results, error) {
	fs, err := pData.getFileSystem(logger)
	if err != nil {
		return nil, err
	}
	fs.TotalDataBytes = estimateFsUsage(fs)
	pData.fileSystem = fs
	resultsChannel := make(chan TargetResult, 1)
	numTargets, err := forEachAccountAndRegion(accountProfileNames, regions,
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
	target := Target{AccountName: accountProfileName, Region: region}
	// TODO(rgooch): Move this skip logic to forEachAccountAndRegion().
	if _, ok := pData.skipTargets[target]; ok {
		logger.Println("skipping target")
		channel <- TargetResult{}
		return
	}
	resultMsg := TargetResult{Target: target}
	if snap, ami, err := pData.publishToTarget(awsService, logger); err != nil {
		resultMsg.Error = err
		channel <- resultMsg
	} else {
		resultMsg.SnapshotId = snap
		resultMsg.AmiId = ami
		channel <- resultMsg
	}
}

func (pData *publishData) publishToTarget(awsService *ec2.EC2,
	logger log.Logger) (string, string, error) {
	unpackerInstances, err := getInstances(awsService, "ImageUnpacker")
	if err != nil {
		return "", "", err
	}
	var unpackerInstance *ec2.Instance
	for _, instance := range unpackerInstances {
		unpackerInstance = instance
	}
	if unpackerInstance == nil {
		return "", "", errors.New("no ImageUnpacker instances found")
	}
	address := *unpackerInstance.PrivateIpAddress + ":" +
		strconv.Itoa(constants.ImageUnpackerPortNumber)
	logger.Printf("Discovered unpacker: %s at %s\n",
		*unpackerInstance.InstanceId, address)
	srpcClient, err := srpc.DialHTTP("tcp", address, time.Second*15)
	if err != nil {
		return "", "", err
	}
	defer srpcClient.Close()
	logger.Printf("Preparing to unpack: %s\n", pData.streamName)
	uclient.PrepareForUnpack(srpcClient, pData.streamName, true, false)
	minBytes := pData.fileSystem.TotalDataBytes + pData.minFreeBytes
	status, err := selectVolume(srpcClient, awsService, pData.streamName,
		minBytes, pData.tags, unpackerInstance, logger)
	if err != nil {
		return "", "", err
	}
	volumeId := status.ImageStreams[pData.streamName].DeviceId
	if status.ImageStreams[pData.streamName].Status !=
		proto.StatusStreamScanned {
		logger.Printf("Preparing to unpack again: %s\n", pData.streamName)
		err := uclient.PrepareForUnpack(srpcClient, pData.streamName, true,
			false)
		if err != nil {
			return "", "", err
		}
	}
	logger.Printf("Unpacking: %s\n", pData.streamName)
	err = uclient.UnpackImage(srpcClient, pData.streamName, pData.imageLeafName)
	if err != nil {
		return "", "", err
	}
	logger.Printf("Capturing: %s\n", pData.streamName)
	err = uclient.PrepareForCapture(srpcClient, pData.streamName)
	if err != nil {
		return "", "", err
	}
	imageName := path.Join(pData.streamName, path.Base(pData.imageLeafName))
	snapshotId, err := createSnapshot(awsService, volumeId, imageName,
		pData.tags, logger)
	if err != nil {
		return "", "", err
	}
	// Kick off scan for next time.
	err = uclient.PrepareForUnpack(srpcClient, pData.streamName, false, true)
	if err != nil {
		return "", "", err
	}
	logger.Println("Registering AMI...")
	amiId, err := registerAmi(awsService, snapshotId, pData.amiName, imageName,
		pData.tags, logger)
	return snapshotId, amiId, nil
}

func estimateFsUsage(fs *filesystem.FileSystem) uint64 {
	var totalDataBytes uint64
	for _, inode := range fs.InodeTable {
		if _, ok := inode.(*filesystem.DirectoryInode); ok {
			totalDataBytes += 1 << 12
		}
		if inode, ok := inode.(*filesystem.RegularInode); ok {
			// Round up to the nearest page size.
			size := (inode.Size >> 12) << 12
			if size < inode.Size {
				size += 1 << 12
			}
			totalDataBytes += size
		}
		if _, ok := inode.(*filesystem.SymlinkInode); ok {
			totalDataBytes += 1 << 9
		}
	}
	return totalDataBytes
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
		return attachVolume(awsService, *instance.InstanceId, volumeId, logger)
	})
	if err != nil {
		return "", err
	}
	return volumeId, nil
}
