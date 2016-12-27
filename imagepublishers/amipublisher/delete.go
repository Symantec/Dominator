package amipublisher

import (
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/log/prefixlogger"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func deleteResources(resources []Resource, logger log.Logger) error {
	sessions := make(map[string]*session.Session)
	services := make(map[Target]*ec2.EC2)
	var firstError error
	// First unregister AMIs.
	for _, resource := range resources {
		if resource.SnapshotId == "" && resource.AmiId == "" {
			continue
		}
		logger := prefixlogger.New(
			resource.AccountName+": "+resource.Region+": ", logger)
		session := sessions[resource.AccountName]
		if session == nil {
			var err error
			if session, err = createSession(resource.AccountName); err != nil {
				return err
			}
			sessions[resource.AccountName] = session
		}
		target := Target{resource.AccountName, resource.Region}
		service := services[target]
		if service == nil {
			service = createService(session, resource.Region)
			services[target] = service
		}
		if resource.AmiId != "" {
			if err := deregisterAmi(service, resource.AmiId); err != nil {
				logger.Printf("error deleting: %s: %s\n", resource.AmiId, err)
				if firstError == nil {
					firstError = err
				}
			} else {
				logger.Printf("deleted: %s\n", resource.AmiId)
			}
		}
	}
	// Now delete snapshots.
	for _, resource := range resources {
		if resource.SnapshotId == "" {
			continue
		}
		service := services[Target{resource.AccountName, resource.Region}]
		if err := deleteSnapshot(service, resource.SnapshotId); err != nil {
			logger.Printf("error deleting: %s: %s\n", resource.SnapshotId, err)
			if firstError == nil {
				firstError = err
			}
		} else {
			logger.Printf("deleted: %s\n", resource.SnapshotId)
		}
	}
	return firstError
}

func deleteTags(resources []Resource, tagKeys []string,
	logger log.Logger) error {
	sessions := make(map[string]*session.Session)
	services := make(map[Target]*ec2.EC2)
	var firstError error
	for _, resource := range resources {
		if resource.SnapshotId == "" && resource.AmiId == "" {
			continue
		}
		session := sessions[resource.AccountName]
		if session == nil {
			var err error
			if session, err = createSession(resource.AccountName); err != nil {
				return err
			}
			sessions[resource.AccountName] = session
		}
		target := Target{resource.AccountName, resource.Region}
		service := services[target]
		if service == nil {
			service = createService(session, resource.Region)
			services[target] = service
		}
		err := deleteTagsFromResources(service, tagKeys, resource.AmiId,
			resource.SnapshotId)
		if err != nil {
			if firstError == nil {
				firstError = err
			}
		}
	}
	return firstError
}
