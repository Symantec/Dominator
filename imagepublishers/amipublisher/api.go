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
	// Computed data follow.
	fileSystem *filesystem.FileSystem
}

type TargetResult struct {
	AccountName string
	Region      string
	SnapshotId  string
	AmiId       string
	Error       error
}

func (v TargetResult) MarshalJSON() ([]byte, error) {
	return v.marshalJSON()
}

type Results []TargetResult

func Publish(imageServerAddress string, streamName string, imageLeafName string,
	minFreeBytes uint64, amiName string, tags map[string]string,
	targetAccountNames []string, targetRegionNames []string,
	logger log.Logger) (Results, error) {
	pData := &publishData{
		imageServerAddress: imageServerAddress,
		streamName:         streamName,
		imageLeafName:      imageLeafName,
		minFreeBytes:       minFreeBytes,
		amiName:            amiName,
		tags:               tags,
	}
	return pData.publish(targetAccountNames, targetRegionNames, logger)
}
