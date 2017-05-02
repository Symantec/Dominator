package main

import (
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/lib/awsutil"
	libjson "github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/mdb"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type awsGeneratorType struct {
	targets        awsutil.TargetList
	filterTagsFile string
}

type resultType struct {
	mdb *mdb.Mdb
	err error
}

type tagFilterType struct {
	Key    string
	Values []string
}

func newAwsGenerator(args []string) (generator, error) {
	return &awsGeneratorType{
			targets: awsutil.TargetList{awsutil.Target{args[1], args[0]}}},
		nil
}

func newAwsFilteredGenerator(args []string) (generator, error) {
	gen := awsGeneratorType{
		filterTagsFile: args[1],
	}
	if err := gen.targets.Set(args[0]); err != nil {
		return nil, err
	}
	return &gen, nil
}

func newAwsLocalGenerator(args []string) (generator, error) {
	region, err := awsutil.GetLocalRegion()
	if err != nil {
		return nil, err
	}
	return &awsGeneratorType{
			targets: awsutil.TargetList{awsutil.Target{"", region}}},
		nil
}

func (g *awsGeneratorType) Generate(unused_datacentre string,
	logger log.Logger) (*mdb.Mdb, error) {
	resultsChannel := make(chan resultType, 1)
	numTargets, err := awsutil.ForEachTarget(g.targets, nil,
		func(awsService *ec2.EC2, account, region string, logger log.Logger) {
			var result resultType
			result.mdb, result.err = g.generateForTarget(awsService, account,
				region, logger)
			resultsChannel <- result
		},
		logger)
	// Collect results.
	var newMdb mdb.Mdb
	hostnames := make(map[string]struct{})
	for i := 0; i < numTargets; i++ {
		result := <-resultsChannel
		if result.err != nil {
			if err == nil {
				err = result.err
				logger.Println(err)
			}
			continue
		}
		for _, machine := range result.mdb.Machines {
			if _, ok := hostnames[machine.Hostname]; ok {
				txt := "duplicate hostname: " + machine.Hostname
				logger.Println(txt)
				if err == nil {
					err = errors.New(txt)
				}
				break
			}
			newMdb.Machines = append(newMdb.Machines, machine)
		}
	}
	return &newMdb, err
}

func (g *awsGeneratorType) generateForTarget(svc *ec2.EC2, accountName string,
	region string, logger log.Logger) (
	*mdb.Mdb, error) {
	filters, err := g.makeFilters()
	if err != nil {
		return nil, err
	}
	resp, err := svc.DescribeInstances(&ec2.DescribeInstancesInput{
		Filters: filters,
	})
	if err != nil {
		return nil, err
	}
	return extractMdb(resp, accountName, region), nil
}

func (g *awsGeneratorType) makeFilters() ([]*ec2.Filter, error) {
	filters := make([]*ec2.Filter, 1, 1)
	filters[0] = &ec2.Filter{
		Name:   aws.String("instance-state-name"),
		Values: []*string{aws.String(ec2.InstanceStateNameRunning)},
	}
	if g.filterTagsFile == "" {
		return filters, nil
	}
	var tags []tagFilterType
	if err := libjson.ReadFromFile(g.filterTagsFile, &tags); err != nil {
		return nil, fmt.Errorf("error loading tags file: %s", err)
	}
	for _, tag := range tags {
		filters = append(filters, &ec2.Filter{
			Name:   aws.String("tag:" + tag.Key),
			Values: aws.StringSlice(tag.Values),
		})
	}
	return filters, nil
}

func extractMdb(output *ec2.DescribeInstancesOutput, accountName string,
	region string) *mdb.Mdb {
	var result mdb.Mdb
	for _, reservation := range output.Reservations {
		accountId := aws.StringValue(reservation.OwnerId)
		for _, instance := range reservation.Instances {
			if instance.PrivateDnsName != nil {
				machine := mdb.Machine{
					Hostname: *instance.PrivateDnsName,
					AwsMetadata: &mdb.AwsMetadata{
						AccountId:   accountId,
						AccountName: accountName,
						InstanceId:  *instance.InstanceId,
						Region:      region,
						Tags:        make(map[string]string),
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
