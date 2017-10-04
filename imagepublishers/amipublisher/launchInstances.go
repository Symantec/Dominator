package amipublisher

import (
	"errors"
	"strconv"
	"time"

	"github.com/Symantec/Dominator/lib/awsutil"
	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func launchInstances(targets awsutil.TargetList, skipList awsutil.TargetList,
	imageSearchTags, vpcSearchTags, subnetSearchTags,
	securityGroupSearchTags awsutil.Tags, instanceType string,
	sshKeyName string, tags map[string]string, replaceInstances bool,
	logger log.Logger) ([]InstanceResult, error) {
	if imageSearchTags["Name"] == "" {
		return nil, errors.New("no image Name search tag")
	}
	resultsChannel := make(chan InstanceResult, 1)
	numTargets, err := awsutil.ForEachTarget(targets, skipList,
		func(awsService *ec2.EC2, account, region string, logger log.Logger) {
			instanceId, privateIp, err := launchInstanceInTarget(awsService,
				imageSearchTags, vpcSearchTags, subnetSearchTags,
				securityGroupSearchTags, instanceType, sshKeyName, tags,
				replaceInstances, logger)
			if err != nil {
				logger.Println(err)
			}
			resultsChannel <- InstanceResult{
				awsutil.Target{account, region},
				instanceId,
				privateIp,
				err,
			}
		},
		logger)
	// Collect results.
	results := make([]InstanceResult, 0, numTargets)
	for i := 0; i < numTargets; i++ {
		result := <-resultsChannel
		if result.AccountName == "" || result.Region == "" {
			continue
		}
		results = append(results, result)
	}
	return results, err
}

func launchInstanceInTarget(awsService *ec2.EC2,
	imageSearchTags, vpcSearchTags, subnetSearchTags,
	securityGroupSearchTags awsutil.Tags,
	instanceType string, sshKeyName string, tags awsutil.Tags,
	replaceInstances bool, logger log.Logger) (string, string, error) {
	oldInstances, err := getInstances(awsService, tags["Name"])
	if err != nil {
		return "", "", err
	}
	if len(oldInstances) > 0 {
		if !replaceInstances {
			return "", "", nil
		}
	}
	image, err := findImage(awsService, imageSearchTags)
	if err != nil {
		return "", "", err
	}
	if image == nil {
		// TODO(rgooch): Create bootstrap image (for unpackers only).
		return "", "", errors.New("no image found")
	}
	instance, err := launchInstance(awsService, image, tags, vpcSearchTags,
		subnetSearchTags, securityGroupSearchTags, instanceType, sshKeyName)
	if err != nil {
		return "", "", err
	}
	instanceId := aws.StringValue(instance.InstanceId)
	privateIp := aws.StringValue(instance.PrivateIpAddress)
	logger.Printf("launched: %s with private IP: %s\n", instanceId, privateIp)
	err = awsService.WaitUntilInstanceRunning(&ec2.DescribeInstancesInput{
		InstanceIds: aws.StringSlice([]string{instanceId}),
	})
	if err != nil {
		if err := libTerminateInstances(awsService, instanceId); err != nil {
			logger.Printf("error terminating: %s\n", err)
		}
		return "", "", err
	}
	logger.Printf("running: %s, connecting...\n", instanceId)
	address := privateIp + ":" + strconv.Itoa(constants.ImageUnpackerPortNumber)
	retryUntil := time.Now().Add(time.Minute * 10)
	srpcClient, err := connectToUnpacker(address, retryUntil, logger)
	if err != nil {
		if err := libTerminateInstances(awsService, instanceId); err != nil {
			logger.Printf("error terminating: %s\n", err)
		}
		return "", "", err
	}
	srpcClient.Close()
	if len(oldInstances) > 0 {
		ids := getInstanceIds(oldInstances)
		if err := libTerminateInstances(awsService, ids...); err != nil {
			logger.Printf("error terminating: %s\n", err)
		} else {
			logger.Println("terminated old instance(s): ", ids)
		}
	}
	return instanceId, privateIp, nil
}

func launchInstancesForImages(resources []Resource,
	vpcSearchTags, subnetSearchTags, securityGroupSearchTags awsutil.Tags,
	instanceType string, sshKeyName string, tags map[string]string,
	logger log.Logger) ([]InstanceResult, error) {
	resultsChannel := make(chan InstanceResult, 1)
	err := forEachResource(resources, false,
		func(session *session.Session, awsService *ec2.EC2, resource Resource,
			logger log.Logger) error {
			instanceId, privateIp, err := launchInstanceForImage(awsService,
				resource, vpcSearchTags, subnetSearchTags,
				securityGroupSearchTags, instanceType, sshKeyName, tags, logger)
			if err != nil {
				logger.Println(err)
			}
			resultsChannel <- InstanceResult{
				awsutil.Target{resource.AccountName, resource.Region},
				instanceId,
				privateIp,
				err,
			}
			return err
		},
		logger)
	// Collect results.
	results := make([]InstanceResult, 0, len(resources))
	for i := 0; i < len(resources); i++ {
		result := <-resultsChannel
		if result.AccountName == "" || result.Region == "" {
			continue
		}
		results = append(results, result)
	}
	return results, err
}

func launchInstanceForImage(awsService *ec2.EC2, resource Resource,
	vpcSearchTags, subnetSearchTags,
	securityGroupSearchTags awsutil.Tags,
	instanceType string, sshKeyName string, tags awsutil.Tags,
	logger log.Logger) (string, string, error) {
	instance, err := launchInstance(awsService,
		&ec2.Image{ImageId: aws.String(resource.AmiId)},
		tags, vpcSearchTags, subnetSearchTags, securityGroupSearchTags,
		instanceType, sshKeyName)
	if err != nil {
		return "", "", err
	}
	instanceId := aws.StringValue(instance.InstanceId)
	privateIp := aws.StringValue(instance.PrivateIpAddress)
	logger.Printf("launched: %s with private IP: %s\n", instanceId, privateIp)
	if err := createTags(awsService, instanceId, tags); err != nil {
		return "", "", nil
	}
	err = awsService.WaitUntilInstanceRunning(&ec2.DescribeInstancesInput{
		InstanceIds: aws.StringSlice([]string{instanceId}),
	})
	if err != nil {
		return "", "", err
	}
	logger.Printf("running: %s: \n", instanceId)
	return instanceId, privateIp, nil
}
