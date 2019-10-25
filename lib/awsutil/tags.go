package awsutil

import (
	libtags "github.com/Cloud-Foundations/Dominator/lib/tags"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func makeFilterFromTag(tag libtags.Tag) *ec2.Filter {
	if tag.Key == "" {
		return nil
	}
	if tag.Value == "" {
		return &ec2.Filter{
			Name:   aws.String("tag-key"),
			Values: aws.StringSlice([]string{tag.Key}),
		}
	} else {
		return &ec2.Filter{
			Name:   aws.String("tag:" + tag.Key),
			Values: aws.StringSlice([]string{tag.Value}),
		}
	}
}

func createTagsFromList(list []*ec2.Tag) libtags.Tags {
	tags := make(libtags.Tags, len(list))
	for _, tag := range list {
		tags[aws.StringValue(tag.Key)] = aws.StringValue(tag.Value)
	}
	return tags
}

func makeFiltersFromTags(tags libtags.Tags) []*ec2.Filter {
	if len(tags) < 1 {
		return nil
	}
	filters := make([]*ec2.Filter, 0, len(tags))
	for key, value := range tags {
		filters = append(filters, makeFilterFromTag(libtags.Tag{key, value}))
	}
	return filters
}
