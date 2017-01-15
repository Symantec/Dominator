package awsutil

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func listRegions(awsService *ec2.EC2) ([]string, error) {
	out, err := awsService.DescribeRegions(&ec2.DescribeRegionsInput{})
	if err != nil {
		return nil, err
	}
	regionNames := make([]string, 0, len(out.Regions))
	for _, region := range out.Regions {
		regionNames = append(regionNames, aws.StringValue(region.RegionName))
	}
	return regionNames, nil
}
