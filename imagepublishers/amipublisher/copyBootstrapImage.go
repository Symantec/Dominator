package amipublisher

import (
	"errors"
	"fmt"
	uclient "github.com/Symantec/Dominator/imageunpacker/client"
	"github.com/Symantec/Dominator/lib/awsutil"
	"github.com/Symantec/Dominator/lib/format"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/imageunpacker"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type targetResult struct {
	awsService *ec2.EC2
	region     string
	logger     log.Logger
	image      *ec2.Image
	instance   *ec2.Instance
	client     *srpc.Client
	status     proto.GetStatusResponse
	mutex      sync.Mutex // Lock everything below.
	prepared   bool
}

func copyBootstrapImage(streamName string, targets awsutil.TargetList,
	skipList awsutil.TargetList, marketplaceImage, marketplaceLoginName string,
	newImageTags awsutil.Tags, unpackerName string,
	vpcSearchTags, subnetSearchTags, securityGroupSearchTags awsutil.Tags,
	instanceType string, sshKeyName string, logger log.Logger) error {
	imageSearchTags := awsutil.Tags{"Name": streamName}
	type resultType struct {
		targetResult
		error error
	}
	resultsChannel := make(chan *resultType, 1)
	numTargets, err := awsutil.ForEachTarget(targets, skipList,
		func(awsService *ec2.EC2, account, region string, logger log.Logger) {
			result, err := probeTarget(awsService, streamName, imageSearchTags,
				unpackerName, logger)
			result.awsService = awsService
			result.region = region
			result.logger = logger
			resultsChannel <- &resultType{targetResult: result, error: err}
		},
		logger)
	// Collect results.
	targetResults := make([]*targetResult, 0, numTargets)
	haveSource := false
	for i := 0; i < numTargets; i++ {
		result := <-resultsChannel
		if result.error != nil {
			if err == nil {
				err = result.error
			}
		} else {
			target := &result.targetResult
			targetResults = append(targetResults, target)
			if target.client != nil {
				if stream, ok := target.status.ImageStreams[streamName]; ok {
					if stream.DeviceId != "" {
						haveSource = true
					}
				}
			}
		}
	}
	if err != nil {
		return err
	}
	if !haveSource {
		for _, target := range targetResults {
			if target.client != nil {
				target.client.Close()
			}
		}
		return errors.New("no source found for: " + streamName)
	}
	errorChannel := make(chan error, 1)
	for _, target := range targetResults {
		go func(target *targetResult) {
			logger := target.logger
			e := target.bootstrap(streamName, targetResults, marketplaceImage,
				marketplaceLoginName, newImageTags, vpcSearchTags,
				subnetSearchTags, securityGroupSearchTags, instanceType,
				sshKeyName, logger)
			if e != nil {
				logger.Println(e)
			}
			errorChannel <- e
		}(target)
	}
	for range targetResults {
		e := <-errorChannel
		if e != nil && err == nil {
			err = e
		}
	}
	for _, target := range targetResults {
		if target.client != nil {
			target.client.Close()
		}
	}
	return err
}

func probeTarget(awsService *ec2.EC2, streamName string,
	imageSearchTags awsutil.Tags, unpackerName string, logger log.Logger) (
	targetResult, error) {
	var result targetResult
	instance, client, err := getWorkingUnpacker(awsService, unpackerName,
		logger)
	if err == nil {
		result.status, err = uclient.GetStatus(client)
		if err == nil {
			result.instance = instance
			result.client = client
		} else {
			client.Close()
		}
	}
	image, err := findImage(awsService, imageSearchTags)
	if err != nil {
		logger.Println(err)
		return result, err
	}
	result.image = image
	return result, nil
}

func (target *targetResult) bootstrap(streamName string,
	targets []*targetResult, marketplaceImage, marketplaceLoginName string,
	newImageTags awsutil.Tags, vpcSearchTags, subnetSearchTags,
	securityGroupSearchTags awsutil.Tags, instanceType string,
	sshKeyName string, logger log.Logger) error {
	if target.image != nil {
		return nil // Already have an image: nothing to copy in here.
	}
	sourceTarget, err := target.getSourceTarget(streamName, targets)
	if err != nil {
		logger.Println(err)
		return err
	}
	awsService := target.awsService
	image, err := findMarketplaceImage(awsService, marketplaceImage)
	if err != nil {
		return err
	}
	if image == nil {
		return errors.New("no marketplace image found")
	}
	instance, err := launchInstance(awsService, image, nil, vpcSearchTags,
		subnetSearchTags, securityGroupSearchTags, instanceType, sshKeyName)
	if err != nil {
		return err
	}
	instanceId := aws.StringValue(instance.InstanceId)
	instanceIP := aws.StringValue(instance.PrivateIpAddress)
	defer libTerminateInstances(awsService, instanceId)
	logger.Printf("launched: %s (%s)\n", instanceId, instanceIP)
	err = awsService.WaitUntilInstanceRunning(&ec2.DescribeInstancesInput{
		InstanceIds: aws.StringSlice([]string{instanceId}),
	})
	if err != nil {
		return err
	}
	logger.Printf("running: %s\n", instanceId)
	sourceDeviceId := sourceTarget.status.ImageStreams[streamName].DeviceId
	sourceDevice := sourceTarget.status.Devices[sourceDeviceId]
	volumeId, err := createVolume(awsService,
		instance.Placement.AvailabilityZone, sourceDevice.Size, nil, logger)
	if err != nil {
		return err
	}
	if err := attachVolume(awsService, instance, volumeId, logger); err != nil {
		deleteVolume(awsService, volumeId)
		return err
	}
	devices, err := getDevices(instance, marketplaceLoginName, logger)
	if err != nil {
		return err
	}
	if len(devices) < 2 {
		return fmt.Errorf("bad device count: %d", len(devices))
	}
	deviceName := devices[1]
	logger.Printf("device: %s\n", deviceName)
	remoteCommand := fmt.Sprintf("sudo chown %s /dev/%s",
		marketplaceLoginName, deviceName)
	cmd := makeSshCmd(instance, marketplaceLoginName, remoteCommand)
	if err := cmd.Run(); err != nil {
		return err
	}
	sshArgs := strings.Join([]string{
		"-o CheckHostIP=no",
		"-o ServerAliveInterval=17",
		"-o StrictHostKeyChecking=no",
		"-o UserKnownHostsFile=/dev/null",
	}, " ")
	destCommand := fmt.Sprintf("gunzip | sudo dd bs=64k of=/dev/%s; sync",
		deviceName)
	sourceCommand := fmt.Sprintf(
		"sudo dd bs=64k if=/dev/%s | gzip | ssh %s %s@%s \"%s\"",
		sourceDevice.DeviceName, sshArgs, marketplaceLoginName,
		instanceIP, destCommand)
	logger.Printf("copying image contents from %s in %s\n",
		aws.StringValue(instance.PrivateIpAddress), sourceTarget.region)
	startTime := time.Now()
	cmd = makeSshCmd(sourceTarget.instance, sshKeyName, sourceCommand)
	if out, err := cmd.CombinedOutput(); err != nil {
		logger.Println(string(out))
		return err
	}
	logger.Printf("copied in %s\n", format.Duration(time.Since(startTime)))
	snapshotId, err := createSnapshot(awsService, volumeId, "bootstrap",
		newImageTags, logger)
	if err != nil {
		return err
	}
	logger.Println("registering AMI...")
	amiId, err := registerAmi(awsService, snapshotId, "", "",
		streamName+"/bootstrap", newImageTags, 0, logger)
	if err != nil {
		deleteSnapshot(awsService, snapshotId)
		return err
	}
	logger.Printf("registered: %s\n", amiId)
	return nil
}

func (target *targetResult) getSourceTarget(streamName string,
	targets []*targetResult) (*targetResult, error) {
	// Find nearest target with an image.
	var sourceTarget *targetResult
	nearness := -1
	for _, remoteTarget := range targets {
		if remoteTarget.client == nil {
			continue
		}
		if streamInfo, ok := remoteTarget.status.ImageStreams[streamName]; !ok {
			continue
		} else if streamInfo.DeviceId == "" {
			continue
		}
		numMatching := getNumMatching(target.region, remoteTarget.region)
		if numMatching > nearness {
			nearness = numMatching
			sourceTarget = remoteTarget
		}
	}
	sourceTarget.mutex.Lock()
	defer sourceTarget.mutex.Unlock()
	if sourceTarget.prepared {
		return sourceTarget, nil
	}
	sourceTarget.logger.Printf("%s preparing for copy\n",
		aws.StringValue(sourceTarget.instance.InstanceId))
	err := uclient.PrepareForCopy(sourceTarget.client, streamName)
	if err != nil {
		return nil, err
	}
	sourceTarget.logger.Println("prepared for copy")
	sourceTarget.prepared = true
	return sourceTarget, nil
}

func getNumMatching(left, right string) int {
	num := 0
	for index := 0; index < len(left) && index < len(right); index++ {
		if left[index] == right[index] {
			num++
		}
	}
	return num
}

func getDevices(instance *ec2.Instance, loginName string, logger log.Logger) (
	[]string, error) {
	stopTime := time.Now().Add(time.Minute * 10)
	showedError := false
	for time.Now().Before(stopTime) {
		cmd := makeSshCmd(instance, loginName, "ls /sys/block")
		output, err := cmd.Output()
		if err != nil {
			if !showedError {
				logger.Println(err)
				showedError = true
			}
			time.Sleep(time.Second * 17)
			continue
		}
		return strings.Split(string(output), "\n"), nil
	}
	return nil, errors.New("timed out SSHing to instance")
}

func makeSshCmd(instance *ec2.Instance, loginName string,
	remoteCommand string) *exec.Cmd {
	return exec.Command("ssh",
		"-A",
		"-o", "CheckHostIP=no",
		"-o", "StrictHostKeyChecking=no",
		"-o", "User="+loginName,
		"-o", "UserKnownHostsFile=/dev/null",
		aws.StringValue(instance.PrivateIpAddress),
		remoteCommand)
}
