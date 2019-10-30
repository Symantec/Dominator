package amipublisher

import (
	"time"

	"github.com/Cloud-Foundations/Dominator/lib/awsutil"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func expireResources(targets awsutil.TargetList, skipList awsutil.TargetList,
	logger log.Logger) error {
	currentTime := time.Now() // Need a common "now" time.
	waitChannel := make(chan struct{})
	numTargets, err := awsutil.ForEachTarget(targets, skipList,
		func(awsService *ec2.EC2, account, region string, logger log.Logger) {
			expireRegionResources(awsService, currentTime, logger)
			waitChannel <- struct{}{}
		},
		logger)
	for i := 0; i < numTargets; i++ {
		<-waitChannel
	}
	return err
}

func expireRegionResources(awsService *ec2.EC2, currentTime time.Time,
	logger log.Logger) {
	images, err := awsService.DescribeImages(&ec2.DescribeImagesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("tag-key"),
				Values: aws.StringSlice([]string{"ExpiresAt"}),
			},
			{
				Name:   aws.String("is-public"),
				Values: aws.StringSlice([]string{"false"}),
			},
		},
	})
	if err == nil {
		for _, image := range images.Images {
			expireImage(awsService, image, currentTime, logger)
		}
	}
	filters := []*ec2.Filter{
		{
			Name:   aws.String("tag-key"),
			Values: aws.StringSlice([]string{"ExpiresAt"}),
		},
	}
	instances, err := awsService.DescribeInstances(&ec2.DescribeInstancesInput{
		Filters: filters,
	})
	if err == nil {
		for _, reservation := range instances.Reservations {
			for _, instance := range reservation.Instances {
				expireInstance(awsService, instance, currentTime, logger)
			}
		}
	}
	snapshots, err := awsService.DescribeSnapshots(&ec2.DescribeSnapshotsInput{
		Filters: filters,
	})
	if err == nil {
		for _, snapshot := range snapshots.Snapshots {
			expireSnapshot(awsService, snapshot, currentTime, logger)
		}
	}
	volumes, err := awsService.DescribeVolumes(&ec2.DescribeVolumesInput{
		Filters: filters,
	})
	if err == nil {
		for _, volume := range volumes.Volumes {
			expireVolume(awsService, volume, currentTime, logger)
		}
	}
}

func expireImage(awsService *ec2.EC2, image *ec2.Image, currentTime time.Time,
	logger log.Logger) {
	if hasExpired(image.Tags, currentTime) {
		imageId := aws.StringValue(image.ImageId)
		if err := deregisterAmi(awsService, imageId); err != nil {
			logger.Printf("error deleting: %s: %s\n", imageId, err)
		} else {
			logger.Printf("deleted: %s\n", imageId)
		}
	}
}

func expireInstance(awsService *ec2.EC2, instance *ec2.Instance,
	currentTime time.Time, logger log.Logger) {
	if aws.StringValue(instance.State.Name) == ec2.InstanceStateNameTerminated {
		return
	}
	if hasExpired(instance.Tags, currentTime) {
		instanceId := aws.StringValue(instance.InstanceId)
		if err := libTerminateInstances(awsService, instanceId); err != nil {
			logger.Printf("error terminating: %s: %s\n", instanceId, err)
		} else {
			logger.Printf("terminated: %s\n", instanceId)
		}
	}
}

func expireSnapshot(awsService *ec2.EC2, snapshot *ec2.Snapshot,
	currentTime time.Time, logger log.Logger) {
	if hasExpired(snapshot.Tags, currentTime) {
		snapshotId := aws.StringValue(snapshot.SnapshotId)
		if err := deleteSnapshot(awsService, snapshotId); err != nil {
			logger.Printf("error deleting: %s: %s\n", snapshotId, err)
		} else {
			logger.Printf("deleted: %s\n", snapshotId)
		}
	}
}

func expireVolume(awsService *ec2.EC2, volume *ec2.Volume,
	currentTime time.Time, logger log.Logger) {
	if hasExpired(volume.Tags, currentTime) {
		volumeId := aws.StringValue(volume.VolumeId)
		if err := deleteVolume(awsService, volumeId); err != nil {
			logger.Printf("error deleting: %s: %s\n", volumeId, err)
		} else {
			logger.Printf("deleted: %s\n", volumeId)
		}
	}
}

func hasExpired(tags []*ec2.Tag, currentTime time.Time) bool {
	for _, tag := range tags {
		if *tag.Key != "ExpiresAt" {
			continue
		}
		expirationTime, err := time.Parse(ExpiresAtFormat, *tag.Value)
		if err != nil {
			continue
		}
		return currentTime.After(expirationTime)
	}
	return false
}
