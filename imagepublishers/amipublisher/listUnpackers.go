package amipublisher

import (
	uclient "github.com/Symantec/Dominator/imageunpacker/client"
	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/format"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"strconv"
	"time"
)

func listUnpackers(targets TargetList, skipList TargetList, name string,
	logger log.Logger) (
	[]TargetUnpackers, error) {
	resultsChannel := make(chan TargetUnpackers, 1)
	numTargets, err := forEachTarget(targets, skipList,
		func(awsService *ec2.EC2, account, region string, logger log.Logger) {
			unpackers, err := listTargetUnpackers(awsService, name, logger)
			if err != nil {
				logger.Println(err)
			}
			resultsChannel <- TargetUnpackers{
				Target{account, region}, unpackers}
		},
		logger)
	// Collect results.
	results := make([]TargetUnpackers, 0, numTargets)
	for i := 0; i < numTargets; i++ {
		result := <-resultsChannel
		if len(result.Unpackers) > 0 {
			results = append(results, result)
		}
	}
	return results, err
}

func listTargetUnpackers(awsService *ec2.EC2, name string, logger log.Logger) (
	[]Unpacker, error) {
	unpackerInstances, err := getInstances(awsService, name)
	if err != nil {
		return nil, err
	}
	if len(unpackerInstances) < 1 {
		return nil, nil
	}
	unpackers := make([]Unpacker, 0, len(unpackerInstances))
	for _, instance := range unpackerInstances {
		unpackers = append(unpackers, Unpacker{
			aws.StringValue(instance.InstanceId),
			aws.StringValue(instance.PrivateIpAddress),
			aws.StringValue(instance.State.Name),
			timeSinceLastUsed(instance),
		})
	}
	return unpackers, nil
}

func timeSinceLastUsed(instance *ec2.Instance) string {
	if aws.StringValue(instance.State.Name) != ec2.InstanceStateNameRunning {
		return ""
	}
	address := *instance.PrivateIpAddress + ":" +
		strconv.Itoa(constants.ImageUnpackerPortNumber)
	srpcClient, err := srpc.DialHTTP("tcp", address, time.Second*5)
	if err != nil {
		return ""
	}
	defer srpcClient.Close()
	status, err := uclient.GetStatus(srpcClient)
	if err != nil {
		return ""
	}
	return format.Duration(status.TimeSinceLastUsed)
}
