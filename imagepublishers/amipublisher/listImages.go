package amipublisher

import (
	"time"

	"github.com/Cloud-Foundations/Dominator/lib/awsutil"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	libtags "github.com/Cloud-Foundations/Dominator/lib/tags"
)

func listImages(targets awsutil.TargetList, skipList awsutil.TargetList,
	searchTags, excludeSearchTags libtags.Tags, minImageAge time.Duration,
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
