package amipublisher

import (
	"github.com/Symantec/Dominator/lib/log"
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
				if firstError == nil {
					firstError = err
				}
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
			if firstError == nil {
				firstError = err
			}
		}
	}
	return firstError
}
