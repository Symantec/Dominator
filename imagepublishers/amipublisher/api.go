package amipublisher

import (
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/log"
)

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
	Target
	SnapshotId string
	AmiId      string
}

type Results []TargetResult

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

type TargetResult struct {
	Target
	SnapshotId string
	AmiId      string
	Size       uint // Size in GiB.
	Error      error
}

type TargetUnpackers struct {
	Target
	Unpackers []Unpacker
}

type Unpacker struct {
	InstanceId string
	IpAddress  string
	State      string
}

func (v TargetResult) MarshalJSON() ([]byte, error) {
	return v.marshalJSON()
}

func DeleteResources(resources []Resource, logger log.Logger) error {
	return deleteResources(resources, logger)
}

func DeleteTags(resources []Resource, tagKeys []string,
	logger log.Logger) error {
	return deleteTags(resources, tagKeys, logger)
}

func ExpireResources(targets TargetList, skipList TargetList,
	logger log.Logger) error {
	return expireResources(targets, skipList, logger)
}

func ListAccountNames() ([]string, error) {
	return listAccountNames()
}

func ListUnpackers(targets TargetList, skipList TargetList, name string,
	logger log.Logger) (
	[]TargetUnpackers, error) {
	return listUnpackers(targets, skipList, name, logger)
}

func PrepareUnpackers(streamName string, targets TargetList,
	skipList TargetList, name string, logger log.Logger) error {
	return prepareUnpackers(streamName, targets, skipList, name, logger)
}

func Publish(imageServerAddress string, streamName string, imageLeafName string,
	minFreeBytes uint64, amiName string, tags map[string]string,
	targets TargetList, skipList TargetList, unpackerName string,
	logger log.Logger) (
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
