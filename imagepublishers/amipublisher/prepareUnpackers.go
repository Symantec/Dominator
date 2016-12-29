package amipublisher

import (
	"errors"
	uclient "github.com/Symantec/Dominator/imageunpacker/client"
	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/log/prefixlogger"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"strconv"
	"time"
)

func prepareUnpackers(streamName string, accountProfileNames []string,
	regions []string, logger log.Logger) error {
	if len(accountProfileNames) < 1 {
		return errors.New("no account names")
	}
	logger.Println("Creating sessions...")
	accountResultsChannel := make(chan accountResult, 1)
	resultsChannel := make(chan error, 1)
	for _, accountProfileName := range accountProfileNames {
		awsSession, err := createSession(accountProfileName)
		if err != nil {
			return err
		}
		go prepareUnpackersInAccount(awsSession, streamName, accountProfileName,
			regions, accountResultsChannel, resultsChannel, logger)
		if err != nil {
			return err
		}
	}
	var numTargets int
	// Collect account results.
	for range accountProfileNames {
		result := <-accountResultsChannel
		if result.err != nil {
			return result.err
		}
		numTargets += result.numRegions
	}
	// Collect results.
	var firstError error
	for i := 0; i < numTargets; i++ {
		err := <-resultsChannel
		if err != nil && firstError == nil {
			firstError = err
		}
	}
	return firstError
}

func prepareUnpackersInAccount(awsSession *session.Session, streamName string,
	accountProfileName string, regions []string,
	accountResultsChannel chan<- accountResult, resultsChannel chan<- error,
	logger log.Logger) {
	aRegionName := "us-east-1"
	var aAwsService *ec2.EC2
	if len(regions) < 1 {
		var err error
		aAwsService := createService(awsSession, aRegionName)
		regions, err = listRegions(aAwsService)
		if err != nil {
			accountResultsChannel <- accountResult{0, err}
			return
		}
	}
	// Start manager for each region.
	numRegions := 0
	for _, region := range regions {
		logger := prefixlogger.New(accountProfileName+": "+region+": ", logger)
		var awsService *ec2.EC2
		if region == aRegionName && aAwsService != nil {
			awsService = aAwsService
		} else {
			awsService = createService(awsSession, region)
		}
		numRegions++
		go func() {
			err := prepareUnpacker(awsService, streamName, logger)
			if err != nil {
				logger.Println(err)
			}
			resultsChannel <- err
		}()
	}
	accountResultsChannel <- accountResult{numRegions, nil}
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
