package amipublisher

import (
	"github.com/Symantec/Dominator/lib/log"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func deleteResources(resources []Resource, logger log.Logger) error {
	return forEachResource(resources, false,
		func(awsService *ec2.EC2, resource Resource, logger log.Logger) error {
			return deleteResource(awsService, resource, logger)
		},
		logger)
}

func deleteResource(awsService *ec2.EC2, resource Resource,
	logger log.Logger) error {
	var firstError error
	if resource.AmiId != "" {
		if err := deregisterAmi(awsService, resource.AmiId); err != nil {
			logger.Printf("error deleting: %s: %s\n", resource.AmiId, err)
			if firstError == nil {
				firstError = err
			}
		} else {
			logger.Printf("deleted: %s\n", resource.AmiId)
		}
	}
	if resource.SnapshotId != "" {
		if err := deleteSnapshot(awsService, resource.SnapshotId); err != nil {
			logger.Printf("error deleting: %s: %s\n", resource.SnapshotId, err)
			if firstError == nil {
				firstError = err
			}
		} else {
			logger.Printf("deleted: %s\n", resource.SnapshotId)
		}
	}
	return firstError
}

func deleteTags(resources []Resource, tagKeys []string,
	logger log.Logger) error {
	return forEachResource(resources, false,
		func(awsService *ec2.EC2, resource Resource, logger log.Logger) error {
			return deleteTagsForResource(awsService, resource, tagKeys, logger)
		},
		logger)
}

func deleteTagsForResource(awsService *ec2.EC2, resource Resource,
	tagKeys []string, logger log.Logger) error {
	err := deleteTagsFromResources(awsService, tagKeys, resource.AmiId,
		resource.SnapshotId)
	if err != nil {
		logger.Println("error deleting tag(s)")
	}
	return err
}
