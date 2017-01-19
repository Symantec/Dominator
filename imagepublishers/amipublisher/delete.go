package amipublisher

import (
	"github.com/Symantec/Dominator/lib/awsutil"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/aws/aws-sdk-go/aws"
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

func deleteTagsOnUnpackers(targets awsutil.TargetList,
	skipList awsutil.TargetList, name string, tagKeys []string,
	logger log.Logger) error {
	if len(tagKeys) < 1 {
		return nil
	}
	resultsChannel := make(chan error, 1)
	numTargets, err := awsutil.ForEachTarget(targets, skipList,
		func(awsService *ec2.EC2, account, region string, logger log.Logger) {
			err := deleteTagsInTarget(awsService, name, tagKeys, logger)
			if err != nil {
				logger.Println(err)
			}
			resultsChannel <- err
		},
		logger)
	// Collect results.
	for i := 0; i < numTargets; i++ {
		e := <-resultsChannel
		if e != nil && err == nil {
			err = e
		}
	}
	return err
}

func deleteTagsInTarget(awsService *ec2.EC2, name string, tagKeys []string,
	logger log.Logger) error {
	instances, err := getInstances(awsService, name)
	if err != nil {
		return err
	}
	if len(instances) < 1 {
		return nil
	}
	resourceIds := make([]string, 0, len(instances))
	for _, instance := range instances {
		resourceIds = append(resourceIds, aws.StringValue(instance.InstanceId))
	}
	return deleteTagsFromResources(awsService, tagKeys, resourceIds...)
}
