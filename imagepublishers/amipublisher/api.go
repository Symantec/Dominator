package amipublisher

import (
	"time"

	"github.com/Symantec/Dominator/lib/awsutil"
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/log"
	libtags "github.com/Symantec/Dominator/lib/tags"
)

const ExpiresAtFormat = "2006-01-02 15:04:05"

type Image struct {
	awsutil.Target
	AmiId        string
	AmiName      string
	CreationDate string
	Description  string
	Size         uint // Size in GiB.
	Tags         libtags.Tags
}

type Instance struct {
	awsutil.Target
	AmiId      string
	InstanceId string
	LaunchTime string
	Tags       libtags.Tags
}

type InstanceResult struct {
	awsutil.Target
	InstanceId string
	PrivateIp  string
	Error      error
}

func (v InstanceResult) MarshalJSON() ([]byte, error) {
	return v.marshalJSON()
}

type publishData struct {
	imageServerAddress string
	streamName         string
	imageLeafName      string
	minFreeBytes       uint64
	amiName            string
	tags               map[string]string
	unpackerName       string
	s3BucketExpression string
	s3Folder           string
	sharingAccountName string
	publishOptions     *PublishOptions
	// Computed data follow.
	fileSystem *filesystem.FileSystem
}

type PublishOptions struct {
	EnaSupport bool
}

type Resource struct {
	awsutil.Target
	SharedFrom     string
	SnapshotId     string
	S3Bucket       string
	S3ManifestFile string
	AmiId          string
}

type Results []TargetResult

type TargetResult struct {
	awsutil.Target
	SharedFrom     string
	SnapshotId     string
	S3Bucket       string
	S3ManifestFile string
	AmiId          string
	Size           uint // Size in GiB.
	Error          error
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

type UnusedImagesResult struct {
	UnusedImages []Image
	OldInstances []Instance
}

func (v TargetResult) MarshalJSON() ([]byte, error) {
	return v.marshalJSON()
}

func AddVolumes(targets awsutil.TargetList, skipList awsutil.TargetList,
	tags libtags.Tags, unpackerName string, size uint64,
	logger log.Logger) error {
	return addVolumes(targets, skipList, tags, unpackerName, size, logger)
}

func CopyBootstrapImage(streamName string, targets awsutil.TargetList,
	skipList awsutil.TargetList, marketplaceImage, marketplaceLoginName string,
	newImageTags libtags.Tags, unpackerName string,
	vpcSearchTags, subnetSearchTags, securityGroupSearchTags libtags.Tags,
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

func DeleteUnusedImages(targets awsutil.TargetList, skipList awsutil.TargetList,
	searchTags, excludeSearchTags libtags.Tags, minImageAge time.Duration,
	logger log.DebugLogger) (UnusedImagesResult, error) {
	return deleteUnusedImages(targets, skipList, searchTags, excludeSearchTags,
		minImageAge, logger)
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
	securityGroupSearchTags libtags.Tags, instanceType string,
	sshKeyName string, tags map[string]string, replaceInstances bool,
	logger log.Logger) ([]InstanceResult, error) {
	return launchInstances(targets, skipList, imageSearchTags, vpcSearchTags,
		subnetSearchTags, securityGroupSearchTags, instanceType, sshKeyName,
		tags, replaceInstances, logger)
}

func LaunchInstancesForImages(images []Resource,
	vpcSearchTags, subnetSearchTags, securityGroupSearchTags libtags.Tags,
	instanceType string, sshKeyName string, tags map[string]string,
	logger log.Logger) ([]InstanceResult, error) {
	return launchInstancesForImages(images, vpcSearchTags,
		subnetSearchTags, securityGroupSearchTags, instanceType, sshKeyName,
		tags, logger)
}

func ListImages(targets awsutil.TargetList, skipList awsutil.TargetList,
	searchTags, excludeSearchTags libtags.Tags, minImageAge time.Duration,
	logger log.DebugLogger) ([]Image, error) {
	return listImages(targets, skipList, searchTags, excludeSearchTags,
		minImageAge, logger)
}

func ListStreams(targets awsutil.TargetList, skipList awsutil.TargetList,
	unpackerName string, logger log.Logger) (map[string]struct{}, error) {
	return listStreams(targets, skipList, unpackerName, logger)
}

func ListUnpackers(targets awsutil.TargetList, skipList awsutil.TargetList,
	unpackerName string, logger log.Logger) (
	[]TargetUnpackers, error) {
	return listUnpackers(targets, skipList, unpackerName, logger)
}

func ListUnusedImages(targets awsutil.TargetList, skipList awsutil.TargetList,
	searchTags, excludeSearchTags libtags.Tags, minImageAge time.Duration,
	logger log.DebugLogger) (UnusedImagesResult, error) {
	return listUnusedImages(targets, skipList, searchTags, excludeSearchTags,
		minImageAge, logger)
}

func PrepareUnpackers(streamName string, targets awsutil.TargetList,
	skipList awsutil.TargetList, name string, logger log.Logger) error {
	return prepareUnpackers(streamName, targets, skipList, name, logger)
}

func Publish(imageServerAddress string, streamName string, imageLeafName string,
	minFreeBytes uint64, amiName string, tags map[string]string,
	targets awsutil.TargetList, skipList awsutil.TargetList,
	unpackerName string, s3Bucket string, s3Folder string,
	sharingAccountName string, publishOptions PublishOptions,
	logger log.Logger) (Results, error) {
	pData := &publishData{
		imageServerAddress: imageServerAddress,
		streamName:         streamName,
		imageLeafName:      imageLeafName,
		minFreeBytes:       minFreeBytes,
		amiName:            amiName,
		tags:               tags,
		unpackerName:       unpackerName,
		s3BucketExpression: s3Bucket,
		s3Folder:           s3Folder,
		sharingAccountName: sharingAccountName,
		publishOptions:     &publishOptions,
	}
	return pData.publish(targets, skipList, logger)
}

func RemoveUnusedVolumes(targets awsutil.TargetList,
	skipList awsutil.TargetList, unpackerName string, logger log.Logger) error {
	return removeUnusedVolumes(targets, skipList, unpackerName, logger)
}

func SetExclusiveTags(resources []Resource, tagKey string, tagValue string,
	logger log.Logger) error {
	return setExclusiveTags(resources, tagKey, tagValue, logger)
}

func SetTags(targets awsutil.TargetList, skipList awsutil.TargetList,
	name string, tags map[string]string, logger log.Logger) error {
	return setTags(targets, skipList, name, tags, logger)
}

func StartInstances(targets awsutil.TargetList,
	skipList awsutil.TargetList, name string, logger log.Logger) (
	[]InstanceResult, error) {
	return startInstances(targets, skipList, name, logger)
}

func StopIdleUnpackers(targets awsutil.TargetList, skipList awsutil.TargetList,
	name string, idleTimeout time.Duration, logger log.Logger) error {
	return stopIdleUnpackers(targets, skipList, name, idleTimeout, logger)
}

func TerminateInstances(targets awsutil.TargetList,
	skipList awsutil.TargetList, name string, logger log.Logger) error {
	return terminateInstances(targets, skipList, name, logger)
}
