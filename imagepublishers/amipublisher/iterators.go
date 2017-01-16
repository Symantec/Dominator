package amipublisher

import (
	"github.com/Symantec/Dominator/lib/awsutil"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/log/prefixlogger"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func forEachResource(resources []Resource, stopOnError bool,
	resourceFunc func(*ec2.EC2, Resource, log.Logger) error,
	logger log.Logger) error {
	sessions := make(map[string]*session.Session)
	awsServices := make(map[awsutil.Target]*ec2.EC2)
	var firstError error
	for _, resource := range resources {
		session := sessions[resource.AccountName]
		if session == nil {
			var err error
			if session, err = awsutil.CreateSession(
				resource.AccountName); err != nil {
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
		target := awsutil.Target{resource.AccountName, resource.Region}
		awsService := awsServices[target]
		if awsService == nil {
			awsService = awsutil.CreateService(session, resource.Region)
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
