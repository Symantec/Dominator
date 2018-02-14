package awsutil

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func (tag Tag) makeFilter() *ec2.Filter {
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

func createTagsFromList(list []*ec2.Tag) Tags {
	tags := make(Tags, len(list))
	for _, tag := range list {
		tags[aws.StringValue(tag.Key)] = aws.StringValue(tag.Value)
	}
	return tags
}

func (tags Tags) makeFilters() []*ec2.Filter {
	if len(tags) < 1 {
		return nil
	}
	filters := make([]*ec2.Filter, 0, len(tags))
	for key, value := range tags {
		filters = append(filters, Tag{key, value}.makeFilter())
	}
	return filters
}
