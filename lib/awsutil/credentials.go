package awsutil

import (
	"sort"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"os"
	"path"
)

var (
	kDefaultCredentialsOptions = (&CredentialsOptions{}).setDefaults()
)

type sessionResult struct {
	accountName string
	accountId   string
	awsSession  *session.Session
	regions     []string
	err         error
}

func (c *CredentialsOptions) setDefaults() *CredentialsOptions {
	home := os.Getenv("HOME")
	if c.CredentialsPath == "" {
		c.CredentialsPath = path.Join(home, ".aws", "credentials")
	}
	if c.ConfigPath == "" {
		c.ConfigPath = path.Join(home, ".aws", "config")
	}
	return c
}

func tryLoadCredentialsWithOptions(
	options *CredentialsOptions) (*CredentialsStore, map[string]error, error) {
	accountNames, err := listAccountNames(options)
	if err != nil {
		return nil, nil, err
	}
	cs, unloadableAccounts := createCredentials(accountNames, options)
	return cs, unloadableAccounts, nil
}

func loadCredentials() (*CredentialsStore, error) {
	accountNames, err := listAccountNames(kDefaultCredentialsOptions)
	if err != nil {
		return nil, err
	}
	cs, unloadableAccounts := createCredentials(
		accountNames, kDefaultCredentialsOptions)
	for _, err := range unloadableAccounts {
		return nil, err
	}
	return cs, nil
}

func createCredentials(
	accountNames []string, options *CredentialsOptions) (
	*CredentialsStore, map[string]error) {
	cs := &CredentialsStore{
		sessionMap:      make(map[string]*session.Session),
		accountIdToName: make(map[string]string),
		accountNameToId: make(map[string]string),
		accountRegions:  make(map[string][]string),
	}
	resultsChannel := make(chan sessionResult, len(accountNames))
	for _, accountName := range accountNames {
		go func(accountName string) {
			resultsChannel <- createSession(accountName, options)
		}(accountName)
	}
	unloadableAccounts := make(map[string]error)
	for range accountNames {
		result := <-resultsChannel
		if result.err != nil {
			unloadableAccounts[result.accountName] = result.err
		} else {
			cs.accountNames = append(cs.accountNames, result.accountName)
			cs.sessionMap[result.accountName] = result.awsSession
			cs.accountIdToName[result.accountId] = result.accountName
			cs.accountNameToId[result.accountName] = result.accountId
			cs.accountRegions[result.accountName] = result.regions
		}
	}
	close(resultsChannel)
	sort.Strings(cs.accountNames)
	return cs, unloadableAccounts
}

func createSession(
	accountName string, options *CredentialsOptions) sessionResult {
	awsSession, err := session.NewSessionWithOptions(session.Options{
		Profile:           accountName,
		SharedConfigState: session.SharedConfigEnable,
		SharedConfigFiles: []string{
			options.CredentialsPath,
			options.ConfigPath,
		},
	})
	if err != nil {
		return sessionResult{err: err, accountName: accountName}
	}
	stsService := sts.New(awsSession)
	inp := &sts.GetCallerIdentityInput{}
	var accountId string
	if out, err := stsService.GetCallerIdentity(inp); err != nil {
		return sessionResult{err: err, accountName: accountName}
	} else {
		if arnV, err := arn.Parse(aws.StringValue(out.Arn)); err != nil {
			return sessionResult{err: err, accountName: accountName}
		} else {
			accountId = arnV.AccountID
		}
	}
	regions, err := listRegions(CreateService(awsSession, "us-east-1"))
	if err != nil {
		return sessionResult{err: err, accountName: accountName}
	}
	return sessionResult{
		accountName: accountName,
		accountId:   accountId,
		awsSession:  awsSession,
		regions:     regions,
	}
}
