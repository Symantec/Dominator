package amipublisher

import (
	uclient "github.com/Symantec/Dominator/imageunpacker/client"
	"github.com/Symantec/Dominator/lib/awsutil"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func removeUnusedVolumes(targets awsutil.TargetList,
	skipList awsutil.TargetList, unpackerName string, logger log.Logger) error {
	errorsChannel := make(chan error, 1)
	numTargets, err := awsutil.ForEachTarget(targets, skipList,
		func(awsService *ec2.EC2, account, region string, logger log.Logger) {
			removeUnusedVolumesFromTargetWrapper(awsService, unpackerName,
				errorsChannel, logger)
		},
		logger)
	// Collect errors.
	for i := 0; i < numTargets; i++ {
		e := <-errorsChannel
		if e != nil && err != nil {
			err = e
		}
	}
	return err
}

func removeUnusedVolumesFromTargetWrapper(awsService *ec2.EC2,
	unpackerName string, errorChannel chan<- error, logger log.Logger) {
	errorChannel <- removeUnusedVolumesFromTarget(awsService, unpackerName,
		logger)
}

func removeUnusedVolumesFromTarget(awsService *ec2.EC2, unpackerName string,
	logger log.Logger) error {
	unpackerInstance, srpcClient, err := getWorkingUnpacker(awsService,
		unpackerName, logger)
	if err != nil {
		logger.Println(err)
		return err
	}
	defer srpcClient.Close()
	status, err := uclient.GetStatus(srpcClient)
	if err != nil {
		logger.Println(err)
		return err
	}
	volumeIds := make([]string, 0)
	// Remove unused volumes.
	for volumeId, device := range status.Devices {
		if device.StreamName != "" {
			continue
		}
		if err := uclient.RemoveDevice(srpcClient, volumeId); err != nil {
			logger.Println(err)
		} else {
			volumeIds = append(volumeIds, volumeId)
		}
	}
	var firstError error
	instanceId := aws.StringValue(unpackerInstance.InstanceId)
	for _, volumeId := range volumeIds {
		if err := detachVolume(awsService, instanceId, volumeId); err != nil {
			logger.Println(err)
			if firstError == nil {
				firstError = err
			}
			continue
		}
		dvi := &ec2.DescribeVolumesInput{
			VolumeIds: aws.StringSlice([]string{volumeId}),
		}
		if err := awsService.WaitUntilVolumeAvailable(dvi); err != nil {
			logger.Println(err)
			if firstError == nil {
				firstError = err
			}
			continue
		}
		logger.Printf("%s detached from: %s\n", volumeId, instanceId)
		if err := deleteVolume(awsService, volumeId); err != nil {
			logger.Println(err)
			if firstError == nil {
				firstError = err
			}
			continue
		}
		logger.Printf("deleted: %s\n", volumeId)
	}
	return nil
}
