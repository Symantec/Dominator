package amipublisher

import (
	"github.com/Symantec/Dominator/lib/awsutil"
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/log"
	"time"
)

type InstanceResult struct {
	awsutil.Target
	InstanceId string
	Error      error
}

type publishData struct {
	imageServerAddress string
	streamName         string
	imageLeafName      string
	minFreeBytes       uint64
	amiName            string
	tags               map[string]string
	unpackerName       string
	// Computed data follow.
	fileSystem *filesystem.FileSystem
}

type Resource struct {
	awsutil.Target
	SnapshotId string
	AmiId      string
}

type Results []TargetResult

type TargetResult struct {
	awsutil.Target
	SnapshotId string
	AmiId      string
	Size       uint // Size in GiB.
	Error      error
}

type TargetUnpackers struct {
	awsutil.Target
	Unpackers []Unpacker
}

type Unpacker struct {
	InstanceId        string
	IpAddress         string
	State             string
	TimeSinceLastUsed string `json:",omitempty"`
}

func (v TargetResult) MarshalJSON() ([]byte, error) {
	return v.marshalJSON()
}

func CopyBootstrapImage(streamName string, targets awsutil.TargetList,
	skipList awsutil.TargetList, marketplaceImage, marketplaceLoginName string,
	newImageTags awsutil.Tags, unpackerName string,
	vpcSearchTags, subnetSearchTags, securityGroupSearchTags awsutil.Tags,
	instanceType string, sshKeyName string, logger log.Logger) error {
	return copyBootstrapImage(streamName, targets, skipList, marketplaceImage,
		marketplaceLoginName, newImageTags, unpackerName, vpcSearchTags,
		subnetSearchTags, securityGroupSearchTags, instanceType, sshKeyName,
		logger)
}

func DeleteResources(resources []Resource, logger log.Logger) error {
	return deleteResources(resources, logger)
}

func DeleteTags(resources []Resource, tagKeys []string,
	logger log.Logger) error {
	return deleteTags(resources, tagKeys, logger)
}

func DeleteTagsOnUnpackers(targets awsutil.TargetList,
	skipList awsutil.TargetList, name string, tagKeys []string,
	logger log.Logger) error {
	return deleteTagsOnUnpackers(targets, skipList, name, tagKeys, logger)
}

func ExpireResources(targets awsutil.TargetList, skipList awsutil.TargetList,
	logger log.Logger) error {
	return expireResources(targets, skipList, logger)
}

func ImportKeyPair(targets awsutil.TargetList, skipList awsutil.TargetList,
	keyName string, publicKey []byte, logger log.Logger) error {
	return importKeyPair(targets, skipList, keyName, publicKey, logger)
}

func LaunchInstances(targets awsutil.TargetList, skipList awsutil.TargetList,
	imageSearchTags, vpcSearchTags, subnetSearchTags,
	securityGroupSearchTags awsutil.Tags, instanceType string,
	sshKeyName string, tags map[string]string, logger log.Logger) (
	[]InstanceResult, error) {
	return launchInstances(targets, skipList, imageSearchTags, vpcSearchTags,
		subnetSearchTags, securityGroupSearchTags, instanceType, sshKeyName,
		tags, logger)
}

func ListUnpackers(targets awsutil.TargetList, skipList awsutil.TargetList,
	name string, logger log.Logger) (
	[]TargetUnpackers, error) {
	return listUnpackers(targets, skipList, name, logger)
}

func PrepareUnpackers(streamName string, targets awsutil.TargetList,
	skipList awsutil.TargetList, name string, logger log.Logger) error {
	return prepareUnpackers(streamName, targets, skipList, name, logger)
}

func Publish(imageServerAddress string, streamName string, imageLeafName string,
	minFreeBytes uint64, amiName string, tags map[string]string,
	targets awsutil.TargetList, skipList awsutil.TargetList,
	unpackerName string, logger log.Logger) (
	Results, error) {
	pData := &publishData{
		imageServerAddress: imageServerAddress,
		streamName:         streamName,
		imageLeafName:      imageLeafName,
		minFreeBytes:       minFreeBytes,
		amiName:            amiName,
		tags:               tags,
		unpackerName:       unpackerName,
	}
	return pData.publish(targets, skipList, logger)
}

func SetExclusiveTags(resources []Resource, tagKey string, tagValue string,
	logger log.Logger) error {
	return setExclusiveTags(resources, tagKey, tagValue, logger)
}

func SetTags(targets awsutil.TargetList, skipList awsutil.TargetList,
	name string, tags map[string]string, logger log.Logger) error {
	return setTags(targets, skipList, name, tags, logger)
}

func StopIdleUnpackers(targets awsutil.TargetList, skipList awsutil.TargetList,
	name string, idleTimeout time.Duration, logger log.Logger) error {
	return stopIdleUnpackers(targets, skipList, name, idleTimeout, logger)
}

func TerminateInstances(targets awsutil.TargetList,
	skipList awsutil.TargetList, name string, logger log.Logger) error {
	return terminateInstances(targets, skipList, name, logger)
}
