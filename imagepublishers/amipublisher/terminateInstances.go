package amipublisher

import (
	"github.com/Symantec/Dominator/lib/awsutil"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func terminateInstances(targets awsutil.TargetList, skipList awsutil.TargetList,
	name string, logger log.Logger) error {
	resultsChannel := make(chan error, 1)
	numTargets, err := awsutil.ForEachTarget(targets, skipList,
		func(awsService *ec2.EC2, account, region string, logger log.Logger) {
			err := terminateInstancesInTarget(awsService, name, logger)
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

func terminateInstancesInTarget(awsService *ec2.EC2, name string,
	logger log.Logger) error {
	instances, err := getInstances(awsService, name)
	if err != nil {
		return err
	}
	if len(instances) < 1 {
		return nil
	}
	return libTerminateInstances(awsService, getInstanceIds(instances)...)
}
