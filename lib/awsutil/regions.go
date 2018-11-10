package awsutil

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

var (
	metadataServer       = "http://169.254.169.254/"
	instanceDocumentPath = "latest/dynamic/instance-identity/document"
	instanceDocumentMap  map[string]string
)

func getLocalRegion() (string, error) {
	if instanceDocumentMap == nil {
		instanceDocumentMap = make(map[string]string)
	}
	if region, ok := instanceDocumentMap["region"]; ok {
		return region, nil
	}
	resp, err := http.Get(metadataServer + instanceDocumentPath)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", errors.New(resp.Status)
	}
	if body, err := ioutil.ReadAll(resp.Body); err != nil {
		return "", err
	} else {
		value := strings.TrimSpace(string(body))
		if value == "" {
			return "", errors.New("empty body")
		} else {
			data := []byte(value)
			if err := json.Unmarshal(data, &instanceDocumentMap); err != nil {
				return "", err
			}
			if region, ok := instanceDocumentMap["region"]; ok {
				return region, nil
			}
			return "", errors.New("region not found in instance document")
		}
	}

}

func listRegions(awsService *ec2.EC2) ([]string, error) {
	out, err := awsService.DescribeRegions(&ec2.DescribeRegionsInput{})
	if err != nil {
		return nil, fmt.Errorf("ec2.DescribeRegions: %s", err)
	}
	regionNames := make([]string, 0, len(out.Regions))
	for _, region := range out.Regions {
		regionNames = append(regionNames, aws.StringValue(region.RegionName))
	}
	sort.Strings(regionNames)
	return regionNames, nil
}
