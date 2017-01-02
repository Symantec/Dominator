package amipublisher

import (
	"errors"
	uclient "github.com/Symantec/Dominator/imageunpacker/client"
	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/aws/aws-sdk-go/service/ec2"
	"strconv"
	"time"
)

func prepareUnpackers(streamName string, accountProfileNames []string,
	regions []string, logger log.Logger) error {
	resultsChannel := make(chan error, 1)
	numTargets, err := forEachAccountAndRegion(accountProfileNames, regions,
		func(awsService *ec2.EC2, account, region string, logger log.Logger) {
			err := prepareUnpacker(awsService, streamName, logger)
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

func prepareUnpacker(awsService *ec2.EC2, streamName string,
	logger log.Logger) error {
	unpackerInstances, err := getInstances(awsService, "ImageUnpacker")
	if err != nil {
		return err
	}
	var unpackerInstance *ec2.Instance
	for _, instance := range unpackerInstances {
		unpackerInstance = instance
	}
	if unpackerInstance == nil {
		return errors.New("no ImageUnpacker instances found")
	}
	address := *unpackerInstance.PrivateIpAddress + ":" +
		strconv.Itoa(constants.ImageUnpackerPortNumber)
	logger.Printf("discovered unpacker: %s at %s\n",
		*unpackerInstance.InstanceId, address)
	srpcClient, err := srpc.DialHTTP("tcp", address, time.Second*15)
	if err != nil {
		return err
	}
	defer srpcClient.Close()
	return uclient.PrepareForUnpack(srpcClient, streamName, true, false)
}
