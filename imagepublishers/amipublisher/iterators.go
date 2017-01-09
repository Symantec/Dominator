package amipublisher

import (
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

func forEachTarget(targets TargetList,
	targetFunc func(*ec2.EC2, string, string, log.Logger),
	logger log.Logger) (int, error) {
	if len(targets) < 1 { // Full wildcard.
		targets = make(TargetList, 1)
	}
	accountMap := make(map[string][]string) // Key: accountName, value: regions.
	var allAccountNames []string
	// Expand any wildcard account names.
	for _, target := range targets {
		if target.AccountName == "" {
			if allAccountNames == nil {
				var err error
				allAccountNames, err = ListAccountNames()
				if err != nil {
					return 0, err
				}
			}
			for _, accountName := range allAccountNames {
				regions := accountMap[accountName]
				regions = append(regions, target.Region)
				accountMap[accountName] = regions
			}
		} else {
			regions := accountMap[target.AccountName]
			regions = append(regions, target.Region)
			accountMap[target.AccountName] = regions
		}
	}
	logger.Println("Creating sessions...")
	awsSessions := make(map[string]*session.Session, len(accountMap))
	for accountName := range accountMap {
		awsSession, err := createSession(accountName)
		if err != nil {
			return 0, err
		}
		awsSessions[accountName] = awsSession
	}
	logger.Println("Starting goroutines...")
	accountResultsChannel := make(chan resultsType, 1)
	var numTargets int
	for accountName, regions := range accountMap {
		// Remove duplicate/redundant regions.
		regionMap := make(map[string]struct{})
		for _, region := range regions {
			if region == "" {
				regionMap = nil
				break
			}
			regionMap[region] = struct{}{}
		}
		regionList := make([]string, 0, len(regionMap))
		for region := range regionMap {
			regionList = append(regionList, region)
		}
		go forEachRegionInAccount(awsSessions[accountName],
			accountName, regionList, accountResultsChannel, targetFunc,
			logger)
	}
	var firstError error
	// Collect account results.
	for range accountMap {
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
