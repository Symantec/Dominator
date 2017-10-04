package amipublisher

import (
	"errors"
	"os"
	"path"
	"sync"

	iclient "github.com/Symantec/Dominator/imageserver/client"
	uclient "github.com/Symantec/Dominator/imageunpacker/client"
	"github.com/Symantec/Dominator/lib/awsutil"
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/imageunpacker"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type sharingStateType struct {
	sharingAccountName string
	sync.Cond
	sync.Mutex                          // Covers everything below.
	results    map[string]*TargetResult // Key: Region.
	sharers    map[string]*ec2.EC2      // Key: Region.
}

func (pData *publishData) publish(targets awsutil.TargetList,
	skipList awsutil.TargetList, logger log.Logger) (
	Results, error) {
	if pData.sharingAccountName != "" && pData.s3BucketExpression == "" {
		return nil, errors.New("sharing not supported for EBS AMIs")
	}
	fs, err := pData.getFileSystem(logger)
	if err != nil {
		return nil, err
	}
	fs.TotalDataBytes = fs.EstimateUsage(0)
	pData.fileSystem = fs
	resultsChannel := make(chan TargetResult, 1)
	sharingState := makeSharingState(pData.sharingAccountName)
	numTargets, err := awsutil.ForEachTarget(targets, skipList,
		func(awsService *ec2.EC2, account, region string, logger log.Logger) {
			pData.publishToTargetWrapper(awsService, account, region,
				sharingState, resultsChannel, logger)
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
	accountProfileName string, region string, sharingState *sharingStateType,
	channel chan<- TargetResult, logger log.Logger) {
	target := awsutil.Target{AccountName: accountProfileName, Region: region}
	resultMsg := TargetResult{}
	res, err := pData.publishToTarget(awsService, accountProfileName, region,
		sharingState, logger)
	if res != nil {
		resultMsg = *res
	}
	resultMsg.Target = target
	resultMsg.Error = err
	if err != nil {
		logger.Println(err)
	}
	sharingState.publish(awsService, resultMsg)
	channel <- resultMsg
}

func (pData *publishData) publishToTarget(awsService *ec2.EC2,
	accountProfileName string, region string, sharingState *sharingStateType,
	logger log.Logger) (*TargetResult, error) {
	imageName := path.Join(pData.streamName, path.Base(pData.imageLeafName))
	if sharingState != nil &&
		sharingState.sharingAccountName != accountProfileName {
		return sharingState.harvest(awsService, region, imageName, pData.tags,
			logger)
	}
	unpackerInstance, srpcClient, err := getWorkingUnpacker(awsService,
		pData.unpackerName, logger)
	if err != nil {
		return nil, err
	}
	defer srpcClient.Close()
	logger.Printf("Preparing to unpack: %s\n", pData.streamName)
	uclient.PrepareForUnpack(srpcClient, pData.streamName, true, false)
	usageEstimate := pData.fileSystem.EstimateUsage(0)
	minBytes := usageEstimate + usageEstimate>>2 // 25% extra for updating.
	status, err := selectVolume(srpcClient, awsService, pData.streamName,
		minBytes, pData.tags, unpackerInstance, logger)
	if err != nil {
		return nil, err
	}
	volumeId := status.ImageStreams[pData.streamName].DeviceId
	if status.ImageStreams[pData.streamName].Status !=
		proto.StatusStreamScanned {
		logger.Printf("Preparing to unpack again: %s\n", pData.streamName)
		err := uclient.PrepareForUnpack(srpcClient, pData.streamName, true,
			false)
		if err != nil {
			return nil, err
		}
	}
	logger.Printf("Unpacking: %s\n", pData.streamName)
	err = uclient.UnpackImage(srpcClient, pData.streamName, pData.imageLeafName)
	if err != nil {
		return nil, err
	}
	logger.Printf("Preparing to capture: %s\n", pData.streamName)
	err = uclient.PrepareForCapture(srpcClient, pData.streamName)
	if err != nil {
		return nil, err
	}
	var snapshotId string
	logger.Printf("Capturing: %s\n", pData.streamName)
	var s3ManifestFile string
	var s3Manifest string
	s3Bucket := expandBucketName(pData.s3BucketExpression, accountProfileName,
		region)
	if s3Bucket == "" {
		snapshotId, err = createSnapshot(awsService, volumeId, imageName,
			pData.tags, logger)
		if err != nil {
			return nil, err
		}
	} else {
		s3Location := path.Join(s3Bucket, pData.s3Folder, imageName)
		s3ManifestFile = path.Join(pData.s3Folder, imageName,
			"image.manifest.xml")
		s3Manifest = path.Join(s3Bucket, s3ManifestFile)
		logger.Printf("Exporting to S3: %s\n", s3Location)
		err := uclient.ExportImage(srpcClient, pData.streamName, "s3",
			s3Location)
		if err != nil {
			return nil, err
		}
	}
	// Kick off scan for next time.
	err = uclient.PrepareForUnpack(srpcClient, pData.streamName, false, true)
	if err != nil {
		return nil, err
	}
	logger.Printf("Registering AMI from: %s...\n", snapshotId)
	volumeSize := status.Devices[volumeId].Size >> 30
	imageBytes := usageEstimate + pData.minFreeBytes
	imageGiB := imageBytes >> 30
	if imageGiB<<30 < imageBytes {
		imageGiB++
	}
	if volumeSize > imageGiB {
		imageGiB = volumeSize
	}
	amiId, err := registerAmi(awsService, snapshotId, s3Manifest, pData.amiName,
		imageName, pData.tags, imageGiB, logger)
	if err != nil {
		logger.Printf("Error registering AMI: %s\n", err)
	}
	return &TargetResult{
		SnapshotId:     snapshotId,
		S3Bucket:       s3Bucket,
		S3ManifestFile: s3ManifestFile,
		AmiId:          amiId,
		Size:           uint(imageGiB),
	}, err
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
	var oldVolumeId string
	if streamInfo, ok := status.ImageStreams[streamName]; ok {
		if deviceInfo, ok := status.Devices[streamInfo.DeviceId]; ok {
			if minBytes <= deviceInfo.Size {
				return status, nil
			}
			oldVolumeId = streamInfo.DeviceId
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
	if oldVolumeId != "" { // Remove old volume.
		logger.Printf("detaching old volume: %s\n", oldVolumeId)
		if err := uclient.RemoveDevice(srpcClient, oldVolumeId); err != nil {
			return proto.GetStatusResponse{}, err
		}
		instId := aws.StringValue(instance.InstanceId)
		if err := detachVolume(awsService, instId, oldVolumeId); err != nil {
			return proto.GetStatusResponse{}, err
		}
		logger.Printf("deleting old volume: %s\n", oldVolumeId)
		if err := deleteVolume(awsService, oldVolumeId); err != nil {
			return proto.GetStatusResponse{}, err
		}
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

func expandBucketName(expr, accountProfileName, region string) string {
	if expr == "" {
		return ""
	}
	return os.Expand(expr, func(variable string) string {
		if variable == "region" {
			return region
		}
		if variable == "accountName" {
			return accountProfileName
		}
		return variable
	})
}

func makeSharingState(sharingAccountName string) *sharingStateType {
	if sharingAccountName == "" {
		return nil
	}
	sharingState := sharingStateType{sharingAccountName: sharingAccountName}
	sharingState.Cond.L = &sharingState.Mutex
	sharingState.results = make(map[string]*TargetResult)
	sharingState.sharers = make(map[string]*ec2.EC2)
	return &sharingState
}

func (ss *sharingStateType) publish(awsService *ec2.EC2, result TargetResult) {
	if ss == nil {
		return
	}
	if ss.sharingAccountName != result.AccountName {
		return
	}
	ss.Lock()
	defer ss.Unlock()
	ss.results[result.Region] = &result
	ss.sharers[result.Region] = awsService
	ss.Broadcast()
}

func (ss *sharingStateType) harvest(awsService *ec2.EC2, region string,
	imageName string, tags awsutil.Tags, logger log.Logger) (
	*TargetResult, error) {
	ownerId, err := getAccountId(awsService)
	if err != nil {
		return nil, err
	}
	logger.Printf("Waiting to harvest AMI from: %s\n", ss.sharingAccountName)
	ss.Lock()
	defer ss.Unlock()
	for ss.results[region] == nil {
		ss.Wait()
	}
	result := ss.results[region]
	sharerService := ss.sharers[region]
	if result.Error != nil {
		return nil, result.Error
	}
	logger.Printf("Remote AMI ID: %s\n", result.AmiId)
	_, err = sharerService.ModifyImageAttribute(&ec2.ModifyImageAttributeInput{
		ImageId: aws.String(result.AmiId),
		LaunchPermission: &ec2.LaunchPermissionModifications{
			Add: []*ec2.LaunchPermission{
				{
					UserId: aws.String(ownerId),
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}
	tags = tags.Copy()
	tags["Name"] = path.Dir(imageName)
	if err := createTags(awsService, result.AmiId, tags); err != nil {
		return nil, err
	}
	newResult := *result
	newResult.SharedFrom = ss.sharingAccountName
	return &newResult, nil
}
