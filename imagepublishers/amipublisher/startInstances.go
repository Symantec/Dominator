package amipublisher

import (
	"errors"
	"github.com/Symantec/Dominator/lib/awsutil"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func startInstances(targets awsutil.TargetList, skipList awsutil.TargetList,
	name string, logger log.Logger) ([]InstanceResult, error) {
	resultsChannel := make(chan InstanceResult, 1)
	numTargets, err := awsutil.ForEachTarget(targets, skipList,
		func(awsService *ec2.EC2, account, region string, logger log.Logger) {
			instanceId, privateIp, err := startInstancesInTarget(awsService,
				name, logger)
			if err != nil {
				logger.Println(err)
			}
			resultsChannel <- InstanceResult{
				awsutil.Target{account, region},
				instanceId,
				privateIp,
				err,
			}
		},
		logger)
	// Collect results.
	results := make([]InstanceResult, 0, numTargets)
	for i := 0; i < numTargets; i++ {
		result := <-resultsChannel
		if result.AccountName == "" || result.Region == "" {
			continue
		}
		results = append(results, result)
	}
	return results, err
}

func startInstancesInTarget(awsService *ec2.EC2, name string,
	logger log.Logger) (string, string, error) {
	instances, err := getInstances(awsService, name)
	if err != nil {
		return "", "", err
	}
	if len(instances) < 1 {
		return "", "", nil
	}
	if len(instances) > 1 {
		return "", "", errors.New("multiple instances")
	}
	instanceId := getInstanceIds(instances)[0]
	instanceIp := aws.StringValue(instances[0].PrivateIpAddress)
	if aws.StringValue(instances[0].State.Name) ==
		ec2.InstanceStateNameRunning {
		return instanceId, instanceIp, nil
	}
	logger.Printf("starting instance: %s\n", instanceId)
	if err := libStartInstances(awsService, instanceId); err != nil {
		return "", "", err
	}
	return instanceId, instanceIp, nil
}
