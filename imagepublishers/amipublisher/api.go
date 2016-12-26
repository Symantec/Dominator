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
	skipTargets        map[Target]struct{}
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

type TargetResult struct {
	Target
	SnapshotId string
	AmiId      string
	Error      error
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

func ExpireResources(accountNames []string, logger log.Logger) error {
	return expireResources(accountNames, logger)
}

func Publish(imageServerAddress string, streamName string, imageLeafName string,
	minFreeBytes uint64, amiName string, tags map[string]string,
	targetAccountNames []string, targetRegionNames []string,
	skipList []Target, logger log.Logger) (Results, error) {
	skipTargets := make(map[Target]struct{})
	for _, target := range skipList {
		skipTargets[Target{target.AccountName, target.Region}] = struct{}{}
	}
	pData := &publishData{
		imageServerAddress: imageServerAddress,
		streamName:         streamName,
		imageLeafName:      imageLeafName,
		minFreeBytes:       minFreeBytes,
		amiName:            amiName,
		tags:               tags,
		skipTargets:        skipTargets,
	}
	return pData.publish(targetAccountNames, targetRegionNames, logger)
}

func SetExclusiveTags(resources []Resource, tagKey string, tagValue string,
	logger log.Logger) error {
	return setExclusiveTags(resources, tagKey, tagValue, logger)
}
