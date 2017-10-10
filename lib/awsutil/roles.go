package awsutil

import (
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/Symantec/Dominator/lib/json"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
)

type roleConfig struct {
	SourceAccountName string
	AccountId         string
	RoleName          string
}

type rolesList struct {
	list map[string]roleConfig // Key: AccountName.
}

func loadRoles() (*rolesList, error) {
	filename := path.Join(os.Getenv("HOME"), ".aws", "roles")
	roles := &rolesList{}
	if err := json.ReadFromFile(filename, &roles.list); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return roles, nil
}

func (rl *rolesList) getCredentials(accountName string, sess *session.Session) (
	*credentials.Credentials, error) {
	roleConfig, ok := rl.list[accountName]
	if !ok {
		return nil, errors.New("unknown account: " + accountName)
	}
	arn := fmt.Sprintf("arn:aws:iam::%s:role/%s",
		roleConfig.AccountId, roleConfig.RoleName)
	return stscreds.NewCredentials(sess, arn), nil
}
