package awsutil

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

func forEachTarget(targets TargetList, skipList TargetList,
	targetFunc func(*ec2.EC2, string, string, log.Logger),
	logger log.Logger) (int, error) {
	if len(targets) < 1 { // Full wildcard.
		targets = make(TargetList, 1)
	}
	accountMap := make(map[string][]string) // Key: accountName, value: regions.
	var allAccountNames []string
	skipTargets := make(map[Target]struct{})
	for _, target := range skipList {
		if target.AccountName != "" || target.Region != "" {
			skipTargets[Target{target.AccountName, target.Region}] = struct{}{}
		}
	}
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
	awsSessions := make(map[string]*session.Session, len(accountMap))
	for accountName := range accountMap {
		awsSession, err := CreateSession(accountName)
		if err != nil {
			return 0, err
		}
		awsSessions[accountName] = awsSession
	}
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
			accountName, regionList, accountResultsChannel, skipTargets,
			targetFunc, logger)
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
	accountName string, regions []string,
	resultsChannel chan<- resultsType, skipTargets map[Target]struct{},
	targetFunc func(*ec2.EC2, string, string, log.Logger),
	logger log.Logger) {
	aRegionName := "us-east-1"
	var aAwsService *ec2.EC2
	if len(regions) < 1 {
		var err error
		aAwsService := CreateService(awsSession, aRegionName)
		regions, err = listRegions(aAwsService)
		if err != nil {
			resultsChannel <- resultsType{0, err}
			return
		}
	}
	if _, ok := skipTargets[Target{accountName, ""}]; ok {
		logger.Println(accountName + ": skipping account")
		resultsChannel <- resultsType{0, nil}
		return
	}
	// Start goroutine for each target ((account,region) tuple).
	numRegions := 0
	for _, region := range regions {
		logger := prefixlogger.New(accountName+": "+region+": ", logger)
		if _, ok := skipTargets[Target{accountName, region}]; ok {
			logger.Println("skipping target")
			continue
		}
		if _, ok := skipTargets[Target{"", region}]; ok {
			logger.Println("skipping region")
			continue
		}
		var awsService *ec2.EC2
		if region == aRegionName && aAwsService != nil {
			awsService = aAwsService
		} else {
			awsService = CreateService(awsSession, region)
		}
		go targetFunc(awsService, accountName, region, logger)
		numRegions++
	}
	resultsChannel <- resultsType{numRegions, nil}
}
