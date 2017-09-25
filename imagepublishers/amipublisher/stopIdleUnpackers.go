package amipublisher

import (
	"strconv"
	"time"

	uclient "github.com/Symantec/Dominator/imageunpacker/client"
	"github.com/Symantec/Dominator/lib/awsutil"
	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func stopIdleUnpackers(targets awsutil.TargetList, skipList awsutil.TargetList,
	name string, idleTimeout time.Duration, logger log.Logger) error {
	resultsChannel := make(chan error, 1)
	numTargets, err := awsutil.ForEachTarget(targets, skipList,
		func(awsService *ec2.EC2, account, region string, logger log.Logger) {
			err := stopIdleUnpackersInTarget(awsService, name, idleTimeout,
				logger)
			if err != nil {
				logger.Println(err)
			}
			resultsChannel <- err
		},
		logger)
	// Collect results.
	firstError := err
	for i := 0; i < numTargets; i++ {
		err := <-resultsChannel
		if err != nil && firstError == nil {
			firstError = err
		}
	}
	return firstError
}

func stopIdleUnpackersInTarget(awsService *ec2.EC2, name string,
	idleTimeout time.Duration, logger log.Logger) error {
	unpackerInstances, err := getInstances(awsService, name)
	if err != nil {
		return err
	}
	if len(unpackerInstances) < 1 {
		return nil
	}
	var firstError error
	for _, instance := range unpackerInstances {
		err := stopIdleUnpacker(awsService, instance, idleTimeout, logger)
		if err != nil && firstError == nil {
			firstError = err
		}
	}
	return firstError
}

func stopIdleUnpacker(awsService *ec2.EC2, instance *ec2.Instance,
	idleTimeout time.Duration, logger log.Logger) error {
	if aws.StringValue(instance.State.Name) != ec2.InstanceStateNameRunning {
		return nil
	}
	address := *instance.PrivateIpAddress + ":" +
		strconv.Itoa(constants.ImageUnpackerPortNumber)
	srpcClient, err := srpc.DialHTTP("tcp", address, time.Second*5)
	if err != nil {
		return err
	}
	defer srpcClient.Close()
	status, err := uclient.GetStatus(srpcClient)
	if err != nil {
		return err
	}
	if status.TimeSinceLastUsed < idleTimeout {
		return nil
	}
	instanceId := aws.StringValue(instance.InstanceId)
	if err := stopInstances(awsService, instanceId); err != nil {
		return err
	}
	logger.Printf("stopped idle instance: %s\n", instanceId)
	return nil
}
