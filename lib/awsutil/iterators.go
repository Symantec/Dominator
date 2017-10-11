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
	targetFunc func(awsService *ec2.EC2, accountName, regionName string,
		logger log.Logger),
	logger log.Logger) (int, error) {
	cs, err := LoadCredentials()
	if err != nil {
		return 0, err
	}
	return cs.ForEachTarget(targets, skipList,
		func(awsSession *session.Session, accountName, regionName string,
			logger log.Logger) {
			targetFunc(CreateService(awsSession, regionName), accountName,
				regionName, logger)
		},
		false, logger)
}

func (cs *CredentialsStore) forEachTarget(targets TargetList,
	skipList TargetList,
	targetFunc func(awsSession *session.Session, accountName, regionName string,
		logger log.Logger),
	wait bool, logger log.Logger) (int, error) {
	if len(targets) < 1 { // Full wildcard.
		targets = make(TargetList, 1)
	}
	accountMap := make(map[string][]string) // Key: accountName, value: regions.
	skipTargets := make(map[Target]struct{})
	for _, target := range skipList {
		if target.AccountName != "" || target.Region != "" {
			skipTargets[Target{target.AccountName, target.Region}] = struct{}{}
		}
	}
	// Expand any wildcard account names.
	for _, target := range targets {
		if target.AccountName == "" {
			for _, accountName := range cs.ListAccountsWithCredentials() {
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
	accountResultsChannel := make(chan resultsType, 1)
	var numTargets int
	var waitChannel chan struct{}
	if wait {
		waitChannel = make(chan struct{}, 1)
	}
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
		awsSession := cs.GetSessionForAccount(accountName)
		go cs.forEachRegionInAccount(awsSession, accountName, regionList,
			accountResultsChannel, skipTargets, targetFunc, waitChannel, logger)
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
	if waitChannel != nil {
		for count := 0; count < numTargets; count++ {
			<-waitChannel
		}
	}
	return numTargets, firstError
}

func (cs *CredentialsStore) forEachRegionInAccount(awsSession *session.Session,
	accountName string, regions []string,
	resultsChannel chan<- resultsType, skipTargets map[Target]struct{},
	targetFunc func(*session.Session, string, string, log.Logger),
	waitChannel chan<- struct{}, logger log.Logger) {
	if len(regions) < 1 {
		var err error
		regions, err = cs.listRegionsForAccount(accountName)
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
		go func(awsSession *session.Session, accountName, regionName string,
			logger log.Logger) {
			targetFunc(awsSession, accountName, regionName, logger)
			if waitChannel != nil {
				waitChannel <- struct{}{}
			}
		}(awsSession, accountName, region, logger)
		numRegions++
	}
	resultsChannel <- resultsType{numRegions, nil}
}
