package awsutil

import (
	"github.com/Symantec/Dominator/lib/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func CreateService(awsSession *session.Session, regionName string) *ec2.EC2 {
	return ec2.New(awsSession, &aws.Config{Region: aws.String(regionName)})
}

func CreateSession(accountProfileName string) (*session.Session, error) {
	return session.NewSessionWithOptions(session.Options{
		Profile:           accountProfileName,
		SharedConfigState: session.SharedConfigEnable})
}

// CredentialsStore records AWS credentials (IAM users and roles) for multiple
// accounts. The methods are safe to use concurrently.
type CredentialsStore struct {
	accountNames    []string
	sessionMap      map[string]*session.Session // Key: account name.
	accountIdToName map[string]string           // Key: account ID.
	accountNameToId map[string]string           // Key: account name.
	accountRegions  map[string][]string         // Key: account name.
}

// LoadCredentials loads credentials from ~/.aws/credentials and roles from
// ~/.aws/config which may be used later.
func LoadCredentials() (*CredentialsStore, error) {
	return loadCredentials()
}

// AccountIdToName will return an account name given an account ID.
func (cs *CredentialsStore) AccountIdToName(accountId string) string {
	return cs.accountIdToName[accountId]
}

// AccountNameToId will return an account ID given an account name.
func (cs *CredentialsStore) AccountNameToId(accountName string) string {
	return cs.accountNameToId[accountName]
}

// ForEachTarget will iterate over a set of targets ((account,region) tuples)
// and will launch a goroutine calling targetFunc for each target.
// The list of targets to iterate over is given by targets and the list of
// targets to skip is given by skipList. An empty string for .AccountName is
// expanded to all available accounts and an empty string for .Region is
// expanded to all regions for an account.
// The number of goroutines is returned. If wait is true then ForEachTarget will
// wait for all the goroutines to complete, else it is the responsibility of the
// caller to wait for the goroutines to complete.
func (cs *CredentialsStore) ForEachTarget(targets TargetList,
	skipList TargetList,
	targetFunc func(awsSession *session.Session, accountName, regionName string,
		logger log.Logger),
	wait bool, logger log.Logger) (int, error) {
	return cs.forEachTarget(targets, skipList, targetFunc, wait, logger)
}

// GetSessionForAccount will return the session credentials available for an
// account. The name of the account should be given by accountName.
// A *session.Session is returned which may be used to bind to AWS services
// (i.e. EC2).
func (cs *CredentialsStore) GetSessionForAccount(
	accountName string) *session.Session {
	return cs.sessionMap[accountName]
}

// GetEC2Service will get an EC2 service handle for an account and region.
func (cs *CredentialsStore) GetEC2Service(accountName,
	regionName string) *ec2.EC2 {
	return CreateService(cs.GetSessionForAccount(accountName), regionName)
}

// ListAccountsWithCredentials will list all accounts for which credentials
// are available.
func (cs *CredentialsStore) ListAccountsWithCredentials() []string {
	return cs.accountNames
}

// ListRegionsForAccount will return all the regions available to an account.
func (cs *CredentialsStore) ListRegionsForAccount(accountName string) []string {
	return cs.accountRegions[accountName]
}

func ForEachTarget(targets TargetList, skipList TargetList,
	targetFunc func(awsService *ec2.EC2, accountName, regionName string,
		logger log.Logger),
	logger log.Logger) (int, error) {
	return forEachTarget(targets, skipList, targetFunc, logger)
}

func GetLocalRegion() (string, error) {
	return getLocalRegion()
}

func ListAccountNames() ([]string, error) {
	return listAccountNames()
}

func ListRegions(awsService *ec2.EC2) ([]string, error) {
	return listRegions(awsService)
}

type Tag struct {
	Key   string
	Value string
}

func (tag Tag) MakeFilter() *ec2.Filter {
	return tag.makeFilter()
}

func (tag *Tag) String() string {
	return tag.string()
}

func (tag *Tag) Set(value string) error {
	return tag.set(value)
}

type Tags map[string]string // Key: tag key, value: tag value.

func (tags Tags) MakeFilters() []*ec2.Filter {
	return tags.makeFilters()
}

func (tags Tags) Copy() Tags {
	return tags.copy()
}

func (to Tags) Merge(from Tags) {
	to.merge(from)
}

func (tags *Tags) String() string {
	return tags.string()
}

func (tags *Tags) Set(value string) error {
	return tags.set(value)
}

type Target struct {
	AccountName string
	Region      string
}

type TargetList []Target

func (list *TargetList) String() string {
	return list.string()
}

func (list *TargetList) Set(value string) error {
	return list.set(value)
}
