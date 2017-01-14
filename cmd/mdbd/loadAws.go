package main

import (
	"github.com/Symantec/Dominator/lib/mdb"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"log"
)

type awsGeneratorType struct {
	svc *ec2.EC2
}

func newAwsGenerator(
	datacentre, profile string) (
	result *awsGeneratorType, err error) {
	sess, err := session.NewSessionWithOptions(
		session.Options{
			Config:  aws.Config{Region: aws.String(datacentre)},
			Profile: profile,
		})
	if err != nil {
		return
	}
	svc := ec2.New(sess)
	result = &awsGeneratorType{svc: svc}
	return
}

func (g *awsGeneratorType) Generate(
	unused_datacentre string, unused_logger *log.Logger) (
	result *mdb.Mdb, err error) {
	params := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("instance-state-name"),
				Values: []*string{aws.String(ec2.InstanceStateNameRunning)},
			},
		},
	}
	resp, err := g.svc.DescribeInstances(params)
	if err != nil {
		return
	}
	result = extractMdb(resp)
	return
}

func extractMdb(output *ec2.DescribeInstancesOutput) *mdb.Mdb {
	var result mdb.Mdb
	for _, reservation := range output.Reservations {
		for _, instance := range reservation.Instances {
			if instance.PrivateDnsName != nil {
				machine := mdb.Machine{
					Hostname: *instance.PrivateDnsName,
					AwsMetadata: &mdb.AwsMetadata{
						InstanceId: *instance.InstanceId,
						Tags:       make(map[string]string),
					},
				}
				if instance.PrivateIpAddress != nil {
					machine.IpAddress = *instance.PrivateIpAddress
				}
				extractTags(instance.Tags, &machine)
				result.Machines = append(result.Machines, machine)
			}
		}
	}
	return &result
}

func extractTags(tags []*ec2.Tag, machine *mdb.Machine) {
	for _, tag := range tags {
		if tag.Key == nil || tag.Value == nil {
			continue
		}
		machine.AwsMetadata.Tags[*tag.Key] = *tag.Value
		switch *tag.Key {
		case "RequiredImage":
			machine.RequiredImage = *tag.Value
		case "PlannedImage":
			machine.PlannedImage = *tag.Value
		case "DisableUpdates":
			machine.DisableUpdates = true
		case "OwnerGroup":
			machine.OwnerGroup = *tag.Value
		}
	}
}
