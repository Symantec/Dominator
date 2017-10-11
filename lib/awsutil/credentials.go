package awsutil

import (
	"fmt"
	"os"
	"path"
	"sort"

	"github.com/Symantec/Dominator/lib/json"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func loadCredentials() (*CredentialsStore, error) {
	accountNames, err := listAccountNames()
	if err != nil {
		return nil, err
	}
	rolesList, err := loadRoles()
	if err != nil {
		return nil, err
	}
	for accountName := range rolesList {
		accountNames = append(accountNames, accountName)
	}
	sort.Strings(accountNames)
	return &CredentialsStore{
		accountNames:   accountNames,
		rolesList:      rolesList,
		sessionMap:     make(map[string]*session.Session),
		accountRegions: make(map[string][]string),
	}, nil
}

func loadRoles() (map[string]roleConfig, error) {
	filename := path.Join(os.Getenv("HOME"), ".aws", "roles")
	var rolesList map[string]roleConfig
	if err := json.ReadFromFile(filename, &rolesList); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return rolesList, nil
}

func (cs *CredentialsStore) getSessionForAccount(accountName string) (
	*session.Session, error) {
	if awsSession, ok := cs.sessionMap[accountName]; ok {
		return awsSession, nil
	}
	if roleConfig, ok := cs.rolesList[accountName]; ok {
		awsSession, err := cs.getSessionForAccount(roleConfig.SourceAccountName)
		if err != nil {
			return nil, err
		}
		arn := fmt.Sprintf("arn:aws:iam::%s:role/%s",
			roleConfig.AccountId, roleConfig.RoleName)
		credentials := stscreds.NewCredentials(awsSession, arn)
		awsSession, err = session.NewSession(
			aws.NewConfig().WithCredentials(credentials))
		if err != nil {
			return nil, err
		}
		cs.sessionMap[accountName] = awsSession
		return awsSession, nil
	}
	awsSession, err := CreateSession(accountName)
	if err != nil {
		return nil, err
	}
	cs.sessionMap[accountName] = awsSession
	return awsSession, nil
}

func (cs *CredentialsStore) getEC2Service(accountName, regionName string) (
	*ec2.EC2, error) {
	awsSession, err := cs.getSessionForAccount(accountName)
	if err != nil {
		return nil, err
	}
	return CreateService(awsSession, regionName), nil
}

func (cs *CredentialsStore) listAccountsWithCredentials() []string {
	return cs.accountNames
}

func (cs *CredentialsStore) listRegionsForAccount(accountName string) (
	[]string, error) {
	if regions, ok := cs.accountRegions[accountName]; ok {
		return regions, nil
	}
	awsService, err := cs.getEC2Service(accountName, "us-east-1")
	if err != nil {
		return nil, err
	}
	regions, err := listRegions(awsService)
	if err != nil {
		return nil, err
	}
	return regions, nil
}
