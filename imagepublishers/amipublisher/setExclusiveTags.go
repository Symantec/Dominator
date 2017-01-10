package amipublisher

import (
	"fmt"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func setExclusiveTags(resources []Resource, tagKey string, tagValue string,
	logger log.Logger) error {
	return forEachResource(resources, true,
		func(awsService *ec2.EC2, resource Resource, logger log.Logger) error {
			return setExclusiveTagsForTarget(awsService, resource.AmiId,
				tagKey, tagValue, logger)
		},
		logger)
}

func setExclusiveTagsForTarget(awsService *ec2.EC2, amiId string,
	tagKey string, tagValue string, logger log.Logger) error {
	if amiId == "" {
		return nil
	}
	// First extract the value of the Name tag which is common to this stream.
	imageIds := make([]string, 1)
	imageIds[0] = amiId
	images, err := awsService.DescribeImages(&ec2.DescribeImagesInput{
		ImageIds: aws.StringSlice(imageIds),
	})
	if err != nil {
		return err
	}
	var nameTag string
	for _, tag := range images.Images[0].Tags {
		if aws.StringValue(tag.Key) == "Name" {
			nameTag = aws.StringValue(tag.Value)
			break
		}
	}
	if nameTag == "" {
		return fmt.Errorf("no \"Name\" tag for: %s", amiId)
	}
	filters := make([]*ec2.Filter, 2)
	values0 := make([]string, 1)
	values0[0] = nameTag
	filters[0] = &ec2.Filter{
		Name:   aws.String("tag:Name"),
		Values: aws.StringSlice(values0),
	}
	values1 := make([]string, 1)
	values1[0] = tagKey
	filters[1] = &ec2.Filter{
		Name:   aws.String("tag-key"),
		Values: aws.StringSlice(values1),
	}
	images, err = awsService.DescribeImages(&ec2.DescribeImagesInput{
		Filters: filters,
	})
	if err != nil {
		return err
	}
	tagKeys := make([]string, 1)
	tagKeys[0] = tagKey
	tagAlreadyPresent := false
	for _, image := range images.Images {
		imageId := aws.StringValue(image.ImageId)
		if imageId == amiId {
			for _, tag := range image.Tags {
				if aws.StringValue(tag.Key) != tagKey {
					continue
				}
				if aws.StringValue(tag.Value) == tagValue {
					tagAlreadyPresent = true
				}
				break
			}
			continue
		}
		err := deleteTagsFromResources(awsService, tagKeys, imageId)
		if err != nil {
			return err
		}
		logger.Printf("deleted \"%s\" tag from: %s\n", tagKey, imageId)
	}
	if tagAlreadyPresent {
		return nil
	}
	tags := make(map[string]string)
	tags[tagKey] = tagValue
	logger.Printf("adding \"%s\" tag to: %s\n", tagKey, amiId)
	return createTags(awsService, amiId, tags)
}
