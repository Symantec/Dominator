package amipublisher

import (
	uclient "github.com/Cloud-Foundations/Dominator/imageunpacker/client"
	"github.com/Cloud-Foundations/Dominator/lib/awsutil"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	libtags "github.com/Cloud-Foundations/Dominator/lib/tags"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func addVolumes(targets awsutil.TargetList, skipList awsutil.TargetList,
	tags libtags.Tags, unpackerName string, size uint64,
	logger log.Logger) error {
	errorsChannel := make(chan error, 1)
	numTargets, err := awsutil.ForEachTarget(targets, skipList,
		func(awsService *ec2.EC2, account, region string, logger log.Logger) {
			addVolumeToTargetWrapper(awsService, tags, unpackerName, size,
				errorsChannel, logger)
		},
		logger)
	// Collect errors.
	for i := 0; i < numTargets; i++ {
		e := <-errorsChannel
		if e != nil && err == nil {
			err = e
		}
	}
	return err
}

func addVolumeToTargetWrapper(awsService *ec2.EC2, tags libtags.Tags,
	unpackerName string, size uint64, errorChannel chan<- error,
	logger log.Logger) {
	errorChannel <- addVolumeToTarget(awsService, tags, unpackerName, size,
		logger)
}

func addVolumeToTarget(awsService *ec2.EC2, tags libtags.Tags,
	unpackerName string, size uint64, logger log.Logger) error {
	unpackerInstance, srpcClient, err := getWorkingUnpacker(awsService,
		unpackerName, logger)
	if err != nil {
		logger.Println(err)
		return err
	}
	defer srpcClient.Close()
	volumeId, err := addVolume(srpcClient, awsService, size, tags,
		unpackerInstance, logger)
	if err != nil {
		logger.Println(err)
		return err
	}
	status, err := uclient.GetStatus(srpcClient)
	if err != nil {
		logger.Println(err)
		return err
	}
	logger.Printf("%s attached as device: %s\n",
		volumeId, status.Devices[volumeId].DeviceName)
	return nil
}
