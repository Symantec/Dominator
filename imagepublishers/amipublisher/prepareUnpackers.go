package amipublisher

import (
	"errors"
	uclient "github.com/Symantec/Dominator/imageunpacker/client"
	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"strconv"
	"time"
)

func prepareUnpackers(streamName string, targets TargetList,
	skipList TargetList, name string, logger log.Logger) error {
	resultsChannel := make(chan error, 1)
	numTargets, err := forEachTarget(targets, skipList,
		func(awsService *ec2.EC2, account, region string, logger log.Logger) {
			err := prepareUnpacker(awsService, streamName, name, logger)
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

func prepareUnpacker(awsService *ec2.EC2, streamName string, name string,
	logger log.Logger) error {
	_, srpcClient, err := getWorkingUnpacker(awsService, name, logger)
	if err != nil {
		return err
	}
	defer srpcClient.Close()
	logger.Println("preparing unpacker")
	err = uclient.PrepareForUnpack(srpcClient, streamName, true, false)
	if err != nil {
		return err
	}
	logger.Println("prepared unpacker")
	return nil
}

func getWorkingUnpacker(awsService *ec2.EC2, name string, logger log.Logger) (
	*ec2.Instance, *srpc.Client, error) {
	unpackerInstances, err := getInstances(awsService, name)
	if err != nil {
		return nil, nil, err
	}
	if len(unpackerInstances) < 1 {
		return nil, nil, errors.New("no ImageUnpacker instances found")
	}
	unpackerInstance, err := getRunningInstance(awsService, unpackerInstances,
		logger)
	if err != nil {
		return nil, nil, err
	}
	if unpackerInstance == nil {
		return nil, nil, errors.New("no running ImageUnpacker instances found")
	}
	launchTime := aws.TimeValue(unpackerInstance.LaunchTime)
	if launchTime.After(time.Now()) {
		launchTime = time.Now()
	}
	address := *unpackerInstance.PrivateIpAddress + ":" +
		strconv.Itoa(constants.ImageUnpackerPortNumber)
	logger.Printf("discovered unpacker: %s at %s\n",
		*unpackerInstance.InstanceId, address)
	srpcClient, err := connectToUnpacker(address,
		launchTime.Add(time.Minute*10), logger)
	if err != nil {
		return nil, nil, err
	}
	return unpackerInstance, srpcClient, nil
}

func connectToUnpacker(address string, retryUntil time.Time,
	logger log.Logger) (*srpc.Client, error) {
	for {
		srpcClient, err := srpc.DialHTTP("tcp", address, time.Second*15)
		if err == nil {
			return srpcClient, nil
		}
		if time.Now().After(retryUntil) {
			return nil, errors.New("timed out waiting for unpacker to start")
		}
		logger.Println(err)
	}
}
