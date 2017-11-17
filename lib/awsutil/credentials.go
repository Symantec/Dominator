package awsutil

import (
	"os"
	"sort"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"path"
)

type sessionResult struct {
	accountName string
	accountId   string
	awsSession  *session.Session
	regions     []string
	err         error
}

func getCredentialsPath() string {
	return getAwsPath("AWS_CREDENTIAL_FILE", "credentials")
}

func getConfigPath() string {
	return getAwsPath("AWS_CONFIG_FILE", "config")
}

func getAwsPath(environ, fileName string) string {
	value := os.Getenv(environ)
	if value != "" {
		return value
	}
	home := os.Getenv("HOME")
	return path.Join(home, ".aws", fileName)
}

func (c *CredentialsOptions) setDefaults() *CredentialsOptions {
	if c.CredentialsPath == "" {
		c.CredentialsPath = *awsCredentialsFile
	}
	if c.ConfigPath == "" {
		c.ConfigPath = *awsConfigFile
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
	var options CredentialsOptions
	cs, unloadableAccounts, err := tryLoadCredentialsWithOptions(
		options.setDefaults())
	if err != nil {
		return nil, err
	}
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
