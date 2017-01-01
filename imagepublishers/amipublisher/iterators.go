package amipublisher

import (
	"errors"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/log/prefixlogger"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type resultsType struct {
	numTargets int
	err        error
}

func forEachResource(resources []Resource, stopOnError bool,
	resourceFunc func(*ec2.EC2, Resource, log.Logger) error,
	logger log.Logger) error {
	sessions := make(map[string]*session.Session)
	awsServices := make(map[Target]*ec2.EC2)
	var firstError error
	for _, resource := range resources {
		session := sessions[resource.AccountName]
		if session == nil {
			var err error
			if session, err = createSession(resource.AccountName); err != nil {
				if stopOnError {
					return err
				}
				if firstError == nil {
					firstError = err
				}
				continue
			}
			sessions[resource.AccountName] = session
		}
		target := Target{resource.AccountName, resource.Region}
		awsService := awsServices[target]
		if awsService == nil {
			awsService = createService(session, resource.Region)
			awsServices[target] = awsService
		}
		err := resourceFunc(awsService, resource,
			prefixlogger.New(resource.AccountName+": "+resource.Region+": ",
				logger))
		if err != nil {
			if stopOnError {
				return err
			}
			if firstError == nil {
				firstError = err
			}
		}
	}
	return firstError
}

func forEachAccountAndRegion(accountProfileNames []string, regions []string,
	targetFunc func(*ec2.EC2, string, string, log.Logger),
	logger log.Logger) (int, error) {
	if len(accountProfileNames) < 1 {
		return 0, errors.New("no account names")
	}
	logger.Println("Creating sessions...")
	awsSessions := make(map[string]*session.Session, len(accountProfileNames))
	for _, accountProfileName := range accountProfileNames {
		awsSession, err := createSession(accountProfileName)
		if err != nil {
			return 0, err
		}
		awsSessions[accountProfileName] = awsSession
	}
	logger.Println("Starting goroutines...")
	accountResultsChannel := make(chan resultsType, 1)
	var numTargets int
	for _, accountProfileName := range accountProfileNames {
		go forEachRegionInAccount(awsSessions[accountProfileName],
			accountProfileName, regions, accountResultsChannel, targetFunc,
			logger)
	}
	var firstError error
	// Collect account results.
	for range accountProfileNames {
		result := <-accountResultsChannel
		if result.err != nil && firstError == nil {
			firstError = result.err
		}
		numTargets += result.numTargets
	}
	return numTargets, firstError
}

func forEachRegionInAccount(awsSession *session.Session,
	accountProfileName string, regions []string,
	resultsChannel chan<- resultsType,
	targetFunc func(*ec2.EC2, string, string, log.Logger),
	logger log.Logger) {
	aRegionName := "us-east-1"
	var aAwsService *ec2.EC2
	if len(regions) < 1 {
		var err error
		aAwsService := createService(awsSession, aRegionName)
		regions, err = listRegions(aAwsService)
		if err != nil {
			resultsChannel <- resultsType{0, err}
			return
		}
	}
	// Start goroutine for each target ((account,region) tuple).
	for _, region := range regions {
		logger := prefixlogger.New(accountProfileName+": "+region+": ", logger)
		var awsService *ec2.EC2
		if region == aRegionName && aAwsService != nil {
			awsService = aAwsService
		} else {
			awsService = createService(awsSession, region)
		}
		go targetFunc(awsService, accountProfileName, region, logger)
	}
	resultsChannel <- resultsType{len(regions), nil}
}
