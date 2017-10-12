package awsutil

import (
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
)

type sessionResult struct {
	accountName string
	accountId   string
	awsSession  *session.Session
	regions     []string
	err         error
}

func loadCredentials() (*CredentialsStore, error) {
	accountNames, err := listAccountNames()
	if err != nil {
		return nil, err
	}
	return createCredentials(accountNames)
}

func createCredentials(accountNames []string) (*CredentialsStore, error) {
	sort.Strings(accountNames)
	cs := &CredentialsStore{
		accountNames:    accountNames,
		sessionMap:      make(map[string]*session.Session),
		accountIdToName: make(map[string]string),
		accountNameToId: make(map[string]string),
		accountRegions:  make(map[string][]string),
	}
	resultsChannel := make(chan sessionResult, len(accountNames))
	for _, accountName := range accountNames {
		go func(accountName string) {
			resultsChannel <- createSession(accountName)
		}(accountName)
	}
	var firstError error
	for range accountNames {
		result := <-resultsChannel
		if result.err != nil {
			if firstError != nil {
				firstError = result.err
			}
		} else {
			cs.sessionMap[result.accountName] = result.awsSession
			cs.accountIdToName[result.accountId] = result.accountName
			cs.accountNameToId[result.accountName] = result.accountId
			cs.accountRegions[result.accountName] = result.regions
		}
	}
	if firstError != nil {
		return nil, firstError
	}
	return cs, nil
}

func createSession(accountName string) sessionResult {
	awsSession, err := CreateSession(accountName)
	if err != nil {
		return sessionResult{err: err}
	}
	iamService := iam.New(awsSession)
	var accountId string
	if out, err := iamService.GetUser(&iam.GetUserInput{}); err != nil {
		splitError := strings.Fields(err.Error())
		if len(splitError) > 3 && splitError[0] == "AccessDenied:" &&
			splitError[1] == "User:" {
			if arnV, e := arn.Parse(splitError[2]); e != nil {
				return sessionResult{err: err}
			} else {
				accountId = arnV.AccountID
			}
		} else {
			return sessionResult{err: err}
		}
	} else {
		if arnV, err := arn.Parse(aws.StringValue(out.User.Arn)); err != nil {
			return sessionResult{err: err}
		} else {
			accountId = arnV.AccountID
		}
	}
	regions, err := listRegions(CreateService(awsSession, "us-east-1"))
	if err != nil {
		return sessionResult{err: err}
	}
	return sessionResult{
		accountName: accountName,
		accountId:   accountId,
		awsSession:  awsSession,
		regions:     regions,
	}
}
