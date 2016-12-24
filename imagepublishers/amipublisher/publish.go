package amipublisher

import (
	"errors"
	iclient "github.com/Symantec/Dominator/imageserver/client"
	uclient "github.com/Symantec/Dominator/imageunpacker/client"
	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/log/prefixlogger"
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/imageunpacker"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"path"
	"strconv"
	"time"
)

type accountResult struct {
	numRegions int
	err        error
}

func (pData *publishData) publish(accountProfileNames []string,
	regions []string, logger log.Logger) (Results, error) {
	if len(accountProfileNames) < 1 {
		return nil, errors.New("no account names")
	}
	fs, err := pData.getFileSystem(logger)
	if err != nil {
		return nil, err
	}
	fs.TotalDataBytes = estimateFsUsage(fs)
	pData.fileSystem = fs
	logger.Println("Creating sessions...")
	accountResultsChannel := make(chan accountResult, 1)
	resultsChannel := make(chan TargetResult, 1)
	for _, accountProfileName := range accountProfileNames {
		awsSession, err := createSession(accountProfileName)
		if err != nil {
			return nil, err
		}
		go pData.publishToAccount(awsSession, accountProfileName, regions,
			accountResultsChannel, resultsChannel, logger)
		if err != nil {
			return nil, err
		}
	}
	var numTargets int
	// Collect account results.
	for range accountProfileNames {
		result := <-accountResultsChannel
		if result.err != nil {
			return nil, result.err
		}
		numTargets += result.numRegions
	}
	// Collect results.
	results := make(Results, 0, numTargets)
	for i := 0; i < numTargets; i++ {
		results = append(results, <-resultsChannel)
	}
	return results, nil
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

func (pData *publishData) publishToAccount(awsSession *session.Session,
	accountProfileName string, regions []string,
	accountResultsChannel chan<- accountResult,
	resultsChannel chan<- TargetResult, logger log.Logger) {
	aRegionName := "us-east-1"
	var aAwsService *ec2.EC2
	if len(regions) < 1 {
		var err error
		aAwsService := createService(awsSession, aRegionName)
		regions, err = listRegions(aAwsService)
		if err != nil {
			accountResultsChannel <- accountResult{0, err}
			return
		}
	}
	// Start manager for each region.
	numRegions := 0
	for _, region := range regions {
		logger := prefixlogger.New(accountProfileName+": "+region+": ", logger)
		if _, ok := pData.skipTargets[Target{accountProfileName, region}]; ok {
			logger.Println("skipping target")
			continue
		}
		var awsService *ec2.EC2
		if region == aRegionName && aAwsService != nil {
			awsService = aAwsService
		} else {
			awsService = createService(awsSession, region)
		}
		numRegions++
		go pData.publishToTargetWrapper(accountProfileName, region, awsService,
			resultsChannel, logger)
	}
	accountResultsChannel <- accountResult{numRegions, nil}
}

func (pData *publishData) publishToTargetWrapper(accountProfileName string,
	region string, awsService *ec2.EC2, channel chan<- TargetResult,
	logger log.Logger) {
	resultMsg := TargetResult{
		Target: Target{AccountName: accountProfileName, Region: region},
	}
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
	prepareForUnpack(srpcClient, pData.streamName)
	minBytes := pData.fileSystem.TotalDataBytes + pData.minFreeBytes
	status, err := selectVolume(srpcClient, awsService, pData.streamName,
		minBytes, pData.tags, unpackerInstance, logger)
	if err != nil {
		return "", "", err
	}
	volumeId := status.ImageStreams[pData.streamName].DeviceId
	logger.Printf("Preparing to unpack: %s\n", pData.streamName)
	if err := prepareForUnpack(srpcClient, pData.streamName); err != nil {
		return "", "", err
	}
	logger.Printf("Unpacking: %s\n", pData.streamName)
	err = unpack(srpcClient, pData.streamName, pData.imageLeafName)
	if err != nil {
		return "", "", err
	}
	logger.Printf("Capturing: %s\n", pData.streamName)
	if err := prepareForCapture(srpcClient, pData.streamName); err != nil {
		return "", "", err
	}
	imageName := path.Join(pData.streamName, path.Base(pData.imageLeafName))
	snapshotId, err := createSnapshot(awsService, volumeId, imageName,
		pData.tags, logger)
	if err != nil {
		return "", "", err
	}
	// Kick off scan for next time.
	if err := startScan(srpcClient, pData.streamName); err != nil {
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
	status, err := getStatus(srpcClient)
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
			err := associateStreamWithDevice(srpcClient, streamName, deviceId)
			if err != nil {
				return proto.GetStatusResponse{}, err
			}
			return getStatus(srpcClient)
		}
	}
	// Need to attach another volume.
	volumeId, err := addVolume(srpcClient, awsService, minBytes, tags, instance,
		logger)
	if err != nil {
		return proto.GetStatusResponse{}, err
	}
	err = associateStreamWithDevice(srpcClient, streamName, volumeId)
	if err != nil {
		return proto.GetStatusResponse{}, err
	}
	return getStatus(srpcClient)
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
