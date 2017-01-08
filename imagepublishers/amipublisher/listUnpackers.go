package amipublisher

import (
	"github.com/Symantec/Dominator/lib/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func listUnpackers(accountProfileNames []string, regions []string, name string,
	logger log.Logger) ([]TargetUnpackers, error) {
	resultsChannel := make(chan TargetUnpackers, 1)
	numTargets, err := forEachAccountAndRegion(accountProfileNames, regions,
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
		})
	}
	return unpackers, nil
}
