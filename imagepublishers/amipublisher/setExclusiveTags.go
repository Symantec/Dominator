package amipublisher

import (
	"fmt"
	"github.com/Symantec/Dominator/lib/awsutil"
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
	out, err := awsService.DescribeImages(&ec2.DescribeImagesInput{
		ImageIds: aws.StringSlice(imageIds),
	})
	if err != nil {
		return err
	}
	var nameTag string
	for _, tag := range out.Images[0].Tags {
		if aws.StringValue(tag.Key) == "Name" {
			nameTag = aws.StringValue(tag.Value)
			break
		}
	}
	if nameTag == "" {
		return fmt.Errorf("no \"Name\" tag for: %s", amiId)
	}
	images, err := getImages(awsService,
		awsutil.Tags{"Name": nameTag, tagKey: ""})
	if err != nil {
		return err
	}
	tagKeysToStrip := []string{tagKey, "Name"}
	tagAlreadyPresent := false
	for _, image := range images {
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
		err := deleteTagsFromResources(awsService, tagKeysToStrip, imageId)
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
