package amipublisher

import (
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/log/prefixlogger"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"sync"
	"time"
)

const expiresAtFormat = "2006-01-02:15:04:05"

func expireResources(accountNames []string, logger log.Logger) error {
	waitGroup := &sync.WaitGroup{}
	currentTime := time.Now() // Need a common "now" time.
	for _, accountName := range accountNames {
		awsSession, err := createSession(accountName)
		if err != nil {
			return err
		}
		aRegionName := "us-east-1"
		aAwsService := createService(awsSession, aRegionName)
		regions, err := listRegions(aAwsService)
		if err != nil {
			return err
		}
		for _, region := range regions {
			awsService := aAwsService
			if region != aRegionName {
				awsService = createService(awsSession, region)
			}
			logger := prefixlogger.New(accountName+": "+region+": ", logger)
			waitGroup.Add(1)
			go expireRegionResources(awsService, waitGroup, currentTime,
				logger)
		}

	}
	waitGroup.Wait()
	return nil
}

func expireRegionResources(awsService *ec2.EC2, waitGroup *sync.WaitGroup,
	currentTime time.Time, logger log.Logger) {
	defer waitGroup.Done()
	filters := make([]*ec2.Filter, 1)
	values := make([]string, 1)
	values[0] = "ExpiresAt"
	filters[0] = &ec2.Filter{
		Name:   aws.String("tag-key"),
		Values: aws.StringSlice(values),
	}
	images, err := awsService.DescribeImages(&ec2.DescribeImagesInput{
		Filters: filters,
	})
	if err == nil {
		for _, image := range images.Images {
			expireImage(awsService, image, currentTime, logger)
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
}

func expireImage(awsService *ec2.EC2, image *ec2.Image, currentTime time.Time,
	logger log.Logger) {
	if hasExpired(image.Tags, currentTime) {
		err := deregisterAmi(awsService, aws.StringValue(image.ImageId))
		if err != nil {
			logger.Printf("error deleting: %s: %s\n", *image.ImageId, err)
		} else {
			logger.Printf("deleted: %s\n", *image.ImageId)
		}
	}
}

func expireSnapshot(awsService *ec2.EC2, snapshot *ec2.Snapshot,
	currentTime time.Time, logger log.Logger) {
	if hasExpired(snapshot.Tags, currentTime) {
		err := deleteSnapshot(awsService, aws.StringValue(snapshot.SnapshotId))
		if err != nil {
			logger.Printf("error deleting: %s: %s\n", *snapshot.SnapshotId, err)
		} else {
			logger.Printf("deleted: %s\n", *snapshot.SnapshotId)
		}
	}
}

func hasExpired(tags []*ec2.Tag, currentTime time.Time) bool {
	for _, tag := range tags {
		if *tag.Key != "ExpiresAt" {
			continue
		}
		expirationTime, err := time.Parse(expiresAtFormat, *tag.Value)
		if err != nil {
			continue
		}
		return currentTime.After(expirationTime)
	}
	return false
}
