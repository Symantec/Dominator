package amipublisher

import (
	"time"

	"github.com/Symantec/Dominator/lib/awsutil"
	"github.com/Symantec/Dominator/lib/log"
)

func listImages(targets awsutil.TargetList, skipList awsutil.TargetList,
	searchTags, excludeSearchTags awsutil.Tags, minImageAge time.Duration,
	logger log.DebugLogger) ([]Image, error) {
	logger.Debugln(0, "loading credentials")
	cs, err := awsutil.LoadCredentials()
	if err != nil {
		return nil, err
	}
	rawResults, err := listUnusedImagesCS(targets, skipList, searchTags,
		excludeSearchTags, minImageAge, cs, true, logger)
	if err != nil {
		return nil, err
	}
	return generateResults(rawResults, logger).UnusedImages, nil
}
