package amipublisher

import (
	"time"

	"github.com/Symantec/Dominator/lib/awsutil"
	"github.com/Symantec/Dominator/lib/concurrent"
	"github.com/Symantec/Dominator/lib/format"
	"github.com/Symantec/Dominator/lib/log"
	libtags "github.com/Symantec/Dominator/lib/tags"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type imageUsage struct {
	images       map[string]*ec2.Image // Key: AMI ID.
	used         usedImages
	oldInstances []*ec2.Instance
}

type targetImageUsage struct {
	accountName string
	region      string
	err         error
	imageUsage
}

type usedImages map[string]struct{} // Key: AMI ID.

func deleteUnusedImages(targets awsutil.TargetList, skipList awsutil.TargetList,
	searchTags, excludeSearchTags libtags.Tags, minImageAge time.Duration,
	logger log.DebugLogger) (UnusedImagesResult, error) {
	logger.Debugln(0, "loading credentials")
	cs, err := awsutil.LoadCredentials()
	if err != nil {
		return UnusedImagesResult{}, err
	}
	rawResults, err := listUnusedImagesCS(targets, skipList, searchTags,
		excludeSearchTags, minImageAge, cs, false, logger)
	if err != nil {
		return UnusedImagesResult{}, err
	}
	concurrentState := concurrent.NewState(0)
	for _, result := range rawResults {
		for amiId, image := range result.images {
			accountName := result.accountName
			region := result.region
			image := image
			err := concurrentState.GoRun(func() error {
				return deleteImage(cs, accountName, region, image)
			})
			if err != nil {
				return UnusedImagesResult{}, err
			}
			logger.Printf("%s: %s: deleted: %s\n",
				result.accountName, result.region, amiId)
		}
	}
	if err := concurrentState.Reap(); err != nil {
		return UnusedImagesResult{}, err
	}
	return generateResults(rawResults, logger), nil
}

func generateResults(rawResults []targetImageUsage,
	logger log.DebugLogger) UnusedImagesResult {
	logger.Debugln(0, "generating results")
	results := UnusedImagesResult{}
	for _, result := range rawResults {
		for amiId, image := range result.images {
			results.UnusedImages = append(results.UnusedImages, Image{
				Target: awsutil.Target{
					AccountName: result.accountName,
					Region:      result.region,
				},
				AmiId:        amiId,
				AmiName:      aws.StringValue(image.Name),
				CreationDate: aws.StringValue(image.CreationDate),
				Description:  aws.StringValue(image.Description),
				Size:         uint(computeImageConsumption(image)),
				Tags:         awsutil.CreateTagsFromList(image.Tags),
			})
		}
		for _, instance := range result.oldInstances {
			results.OldInstances = append(results.OldInstances, Instance{
				Target: awsutil.Target{
					AccountName: result.accountName,
					Region:      result.region,
				},
				AmiId:      aws.StringValue(instance.ImageId),
				InstanceId: aws.StringValue(instance.InstanceId),
				LaunchTime: instance.LaunchTime.Format(
					format.TimeFormatSeconds),
				Tags: awsutil.CreateTagsFromList(instance.Tags),
			})
		}
	}
	return results
}

func listUnusedImages(targets awsutil.TargetList, skipList awsutil.TargetList,
	searchTags, excludeSearchTags libtags.Tags, minImageAge time.Duration,
	logger log.DebugLogger) (UnusedImagesResult, error) {
	logger.Debugln(0, "loading credentials")
	cs, err := awsutil.LoadCredentials()
	if err != nil {
		return UnusedImagesResult{}, err
	}
	rawResults, err := listUnusedImagesCS(targets, skipList, searchTags,
		excludeSearchTags, minImageAge, cs, false, logger)
	if err != nil {
		return UnusedImagesResult{}, err
	}
	return generateResults(rawResults, logger), nil
}

func listUnusedImagesCS(targets awsutil.TargetList, skipList awsutil.TargetList,
	searchTags, excludeSearchTags libtags.Tags, minImageAge time.Duration,
	cs *awsutil.CredentialsStore, ignoreInstances bool,
	logger log.DebugLogger) (
	[]targetImageUsage, error) {
	resultsChannel := make(chan targetImageUsage, 1)
	logger.Debugln(0, "collecting raw data")
	numTargets, err := cs.ForEachEC2Target(targets, skipList,
		func(awsService *ec2.EC2, account, region string, logger log.Logger) {
			usage, err := listTargetUnusedImages(awsService, searchTags,
				excludeSearchTags, cs.AccountNameToId(account), minImageAge,
				ignoreInstances, logger)
			if err != nil {
				logger.Println(err)
			}
			resultsChannel <- targetImageUsage{
				accountName: account,
				region:      region,
				err:         err,
				imageUsage:  usage,
			}
		},
		false, logger)
	if err != nil {
		return nil, err
	}
	// Collect results.
	logger.Debugln(0, "waiting for raw data")
	var firstError error
	rawResults := make([]targetImageUsage, 0, numTargets)
	for i := 0; i < numTargets; i++ {
		result := <-resultsChannel
		if result.err != nil {
			if firstError == nil {
				firstError = result.err
			}
		} else {
			rawResults = append(rawResults, result)
		}
	}
	if firstError != nil {
		return nil, firstError
	}
	// Aggregate used map across accounts.
	logger.Debugln(0, "aggregating usage across accounts")
	imagesUsedPerRegion := make(map[string]usedImages) // Key: region.
	totalImages := 0
	var totalGiBytes int64
	for _, result := range rawResults {
		usedMap := imagesUsedPerRegion[result.region]
		if usedMap == nil {
			usedMap = make(usedImages)
			imagesUsedPerRegion[result.region] = usedMap
		}
		for amiId := range result.used {
			usedMap[amiId] = struct{}{}
		}
		for _, image := range result.images {
			totalGiBytes += computeImageConsumption(image)
		}
		totalImages += len(result.images)
	}
	logger.Printf("total images found: %d consuming %s\n",
		totalImages, format.FormatBytes(uint64(totalGiBytes)<<30))
	if !ignoreInstances {
		// Delete used images from images table.
		logger.Debugln(0, "ignoring used images")
		for _, result := range rawResults {
			usedMap := imagesUsedPerRegion[result.region]
			for amiId := range result.images {
				if _, ok := usedMap[amiId]; ok {
					delete(result.images, amiId)
				}
			}
		}
		// Compute space consumed by unused AMIs.
		numUnusedImages := 0
		var unusedGiBytes int64
		for _, result := range rawResults {
			numUnusedImages += len(result.images)
			for _, image := range result.images {
				unusedGiBytes += computeImageConsumption(image)
			}
		}
		logger.Printf("number of unused images: %d consuming: %s\n",
			numUnusedImages, format.FormatBytes(uint64(unusedGiBytes)<<30))
	}
	return rawResults, nil
}

func listTargetUnusedImages(awsService *ec2.EC2, searchTags libtags.Tags,
	excludeSearchTags libtags.Tags, accountId string,
	minImageAge time.Duration, ignoreInstances bool, logger log.Logger) (
	imageUsage, error) {
	results := imageUsage{
		images: make(map[string]*ec2.Image),
		used:   make(usedImages),
	}
	visibleImages := make(map[string]struct{})
	if images, err := getImages(awsService, "", searchTags); err != nil {
		return imageUsage{}, err
	} else {
		for _, image := range images {
			creationTime, err := time.Parse(creationTimeFormat,
				aws.StringValue(image.CreationDate))
			if err != nil {
				return imageUsage{}, err
			}
			if time.Since(creationTime) < minImageAge {
				continue
			}
			amiId := aws.StringValue(image.ImageId)
			visibleImages[amiId] = struct{}{}
			if aws.StringValue(image.OwnerId) == accountId {
				results.images[amiId] = image
			}
		}
	}
	if len(excludeSearchTags) > 0 {
		images, err := getImages(awsService, accountId, excludeSearchTags)
		if err != nil {
			return imageUsage{}, err
		} else {
			for _, image := range images {
				amiId := aws.StringValue(image.ImageId)
				delete(visibleImages, amiId)
				delete(results.images, amiId)
			}
		}
	}
	if ignoreInstances {
		return results, nil
	}
	instances, err := describeInstances(awsService, nil)
	if err != nil {
		return imageUsage{}, err
	}
	for _, instance := range instances {
		amiId := aws.StringValue(instance.ImageId)
		results.used[amiId] = struct{}{}
		if _, ok := visibleImages[amiId]; ok {
			results.oldInstances = append(results.oldInstances, instance)
		}
	}
	return results, nil
}
