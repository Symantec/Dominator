package amipublisher

import (
	"errors"
	"strconv"
	"time"

	uclient "github.com/Cloud-Foundations/Dominator/imageunpacker/client"
	"github.com/Cloud-Foundations/Dominator/lib/awsutil"
	"github.com/Cloud-Foundations/Dominator/lib/constants"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func prepareUnpackers(streamName string, targets awsutil.TargetList,
	skipList awsutil.TargetList, name string, logger log.Logger) error {
	resultsChannel := make(chan error, 1)
	numTargets, err := awsutil.ForEachTarget(targets, skipList,
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
	if streamName == "" {
		return nil
	}
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
	logger.Printf("discovered unpacker: %s at %s, connecting...\n",
		*unpackerInstance.InstanceId, address)
	retryUntil := launchTime.Add(time.Minute * 10)
	if time.Until(retryUntil) < time.Minute {
		// Give at least one minute grace in case unpacker restarts.
		retryUntil = time.Now().Add(time.Minute)
	}
	srpcClient, err := connectToUnpacker(address, retryUntil, logger)
	if err != nil {
		return nil, nil, err
	}
	return unpackerInstance, srpcClient, nil
}

func connectToUnpacker(address string, retryUntil time.Time,
	logger log.Logger) (*srpc.Client, error) {
	for {
		srpcClient, err := srpc.DialHTTP("tcp", address, time.Second*10)
		if err == nil {
			logger.Printf("connected: %s\n", address)
			return srpcClient, nil
		}
		if time.Now().After(retryUntil) {
			return nil, errors.New("timed out waiting for unpacker to start")
		}
		time.Sleep(time.Second * 5)
	}
}
