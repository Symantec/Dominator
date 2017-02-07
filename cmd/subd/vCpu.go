package main

import (
	"errors"
	"github.com/Symantec/Dominator/lib/log"
	"io/ioutil"
	"net/http"
	"runtime"
	"strings"
)

var metadataServer = "http://169.254.169.254/"
var awsInstanceTypePath = "latest/meta-data/instance-type"

func getAwsInstanceType() (string, error) {
	resp, err := http.Get(metadataServer + awsInstanceTypePath)
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
			return value, nil
		}
	}
}

func getAwsVcpuCreditRate() (uint, error) {
	iType, err := getAwsInstanceType()
	if err != nil {
		return 0, err
	}
	switch iType {
	case "t2.nano":
		return 3, nil
	case "t2.micro":
		return 6, nil
	case "t2.small":
		return 12, nil
	case "t2.medium":
		return 24, nil
	case "t2.large":
		return 36, nil
	case "t2.xlarge":
		return 54, nil
	case "t2.2xlarge":
		return 81, nil
	default:
		return 0, nil
	}
}

func adjustVcpuLimit(limit *uint, logger log.Logger) {
	initialLimit := *limit
	vCpuCreditRate, err := getAwsVcpuCreditRate()
	if err == nil && vCpuCreditRate > 0 {
		newLimit := initialLimit * vCpuCreditRate / 60 / uint(runtime.NumCPU())
		logger.Printf("Adjusting default CPU limit to: %d%%\n", newLimit)
		*limit = newLimit
	}
}
