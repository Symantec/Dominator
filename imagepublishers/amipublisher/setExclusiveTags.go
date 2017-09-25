package amipublisher

import (
	"fmt"

	"github.com/Symantec/Dominator/lib/awsutil"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func setExclusiveTags(resources []Resource, tagKey string, tagValue string,
	logger log.Logger) error {
	return forEachResource(resources, true,
		func(session *session.Session, awsService *ec2.EC2, resource Resource,
			logger log.Logger) error {
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
	out, err := awsService.DescribeImages(&ec2.DescribeImagesInput{
		ImageIds: aws.StringSlice([]string{amiId}),
	})
	if err != nil {
		return err
	}
	if len(out.Images) < 1 {
		return fmt.Errorf("AMI: %s does not exist", amiId)
	}
	var nameTag string
	for _, tag := range out.Images[0].Tags {
		if aws.StringValue(tag.Key) == "Name" {
			nameTag = aws.StringValue(tag.Value)
			break
		}
	}
	if nameTag == "" {
		err := fmt.Errorf("no \"Name\" tag for: %s", amiId)
		logger.Println(err)
		return err
	}
	images, err := getImages(awsService,
		awsutil.Tags{"Name": nameTag, tagKey: ""})
	if err != nil {
		logger.Println(err)
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
			logger.Println(err)
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
