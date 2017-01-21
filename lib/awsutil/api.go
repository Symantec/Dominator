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

func ForEachTarget(targets TargetList, skipList TargetList,
	targetFunc func(*ec2.EC2, string, string, log.Logger),
	logger log.Logger) (int, error) {
	return forEachTarget(targets, skipList, targetFunc, logger)
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

func (tag *Tag) String() string {
	return tag.string()
}

func (tag *Tag) Set(value string) error {
	return tag.set(value)
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
