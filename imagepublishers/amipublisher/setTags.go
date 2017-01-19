package amipublisher

import (
	"github.com/Symantec/Dominator/lib/awsutil"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func setTags(targets awsutil.TargetList, skipList awsutil.TargetList,
	name string, tags map[string]string, logger log.Logger) error {
	resultsChannel := make(chan error, 1)
	numTargets, err := awsutil.ForEachTarget(targets, skipList,
		func(awsService *ec2.EC2, account, region string, logger log.Logger) {
			err := setTagsInTarget(awsService, name, tags, logger)
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

func setTagsInTarget(awsService *ec2.EC2, name string, tags map[string]string,
	logger log.Logger) error {
	unpackerInstances, err := getInstances(awsService, name)
	if err != nil {
		return err
	}
	if len(unpackerInstances) < 1 {
		return nil
	}
	var firstError error
	for _, instance := range unpackerInstances {
		err := createTags(awsService, aws.StringValue(instance.InstanceId),
			tags)
		if err != nil && firstError == nil {
			firstError = err
		}
	}
	return firstError
}
