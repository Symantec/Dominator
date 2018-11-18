package amipublisher

import (
	"github.com/Symantec/Dominator/lib/awsutil"
	"github.com/Symantec/Dominator/lib/format"
	"github.com/Symantec/Dominator/lib/log"
	libtags "github.com/Symantec/Dominator/lib/tags"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func generateUsedResults(rawResults []targetImageUsage,
	logger log.DebugLogger) UsedImagesResult {
	logger.Debugln(0, "generating results")
	results := UsedImagesResult{}
	for _, result := range rawResults {
		for amiId, image := range result.images {
			results.UsedImages = append(results.UsedImages, Image{
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
		for _, instance := range result.allUsingInstances {
			results.UsingInstances = append(results.UsingInstances, Instance{
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

func listUsedImages(targets awsutil.TargetList, skipList awsutil.TargetList,
	searchTags, excludeSearchTags libtags.Tags,
	logger log.DebugLogger) (UsedImagesResult, error) {
	logger.Debugln(0, "loading credentials")
	cs, err := awsutil.LoadCredentials()
	if err != nil {
		return UsedImagesResult{}, err
	}
	rawResults, err := listUsedImagesCS(targets, skipList, searchTags,
		excludeSearchTags, cs, logger)
	if err != nil {
		return UsedImagesResult{}, err
	}
	return generateUsedResults(rawResults, logger), nil
}

func listUsedImagesCS(targets awsutil.TargetList, skipList awsutil.TargetList,
	searchTags, excludeSearchTags libtags.Tags, cs *awsutil.CredentialsStore,
	logger log.DebugLogger) ([]targetImageUsage, error) {
	resultsChannel := make(chan targetImageUsage, 1)
	logger.Debugln(0, "collecting raw data")
	numTargets, err := cs.ForEachEC2Target(targets, skipList,
		func(awsService *ec2.EC2, account, region string, logger log.Logger) {
			usage, err := listTargetUsedImages(awsService, searchTags,
				excludeSearchTags, cs.AccountNameToId(account), logger)
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
	totalInstances := 0
	totalUsingInstances := 0
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
		totalInstances += len(result.allInstances)
		totalUsingInstances += len(result.allUsingInstances)
	}
	logger.Printf("total images found: %d consuming %s\n",
		totalImages, format.FormatBytes(uint64(totalGiBytes)<<30))
	logger.Printf("instances using images: %d/%d\n",
		totalUsingInstances, totalInstances)
	return rawResults, nil
}

func listTargetUsedImages(awsService *ec2.EC2, searchTags libtags.Tags,
	excludeSearchTags libtags.Tags, accountId string,
	logger log.Logger) (imageUsage, error) {
	results := imageUsage{
		images: make(map[string]*ec2.Image),
		used:   make(usedImages),
	}
	visibleImages := make(map[string]struct{})
	if images, err := getImages(awsService, "", searchTags); err != nil {
		return imageUsage{}, err
	} else {
		for _, image := range images {
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
	instances, err := describeInstances(awsService, nil)
	if err != nil {
		return imageUsage{}, err
	}
	results.allInstances = instances
	for _, instance := range instances {
		amiId := aws.StringValue(instance.ImageId)
		results.used[amiId] = struct{}{}
		if _, ok := visibleImages[amiId]; ok {
			results.allUsingInstances = append(results.allUsingInstances,
				instance)
		}
	}
	return results, nil
}
