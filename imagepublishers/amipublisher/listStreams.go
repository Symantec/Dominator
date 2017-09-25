package amipublisher

import (
	"io/ioutil"
	stdlog "log"

	uclient "github.com/Symantec/Dominator/imageunpacker/client"
	"github.com/Symantec/Dominator/lib/awsutil"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/aws/aws-sdk-go/service/ec2"
)

var nullLogger = stdlog.New(ioutil.Discard, "", 0)

func listStreams(targets awsutil.TargetList, skipList awsutil.TargetList,
	name string, logger log.Logger) (
	map[string]struct{}, error) {
	resultsChannel := make(chan map[string]struct{}, 1)
	numTargets, err := awsutil.ForEachTarget(targets, skipList,
		func(awsService *ec2.EC2, account, region string, logger log.Logger) {
			results, err := listTargetStreams(awsService, name, logger)
			if err != nil {
				logger.Println(err)
			}
			resultsChannel <- results
		},
		logger)
	// Collect results.
	results := make(map[string]struct{})
	for i := 0; i < numTargets; i++ {
		result := <-resultsChannel
		for stream := range result {
			results[stream] = struct{}{}
		}
	}
	return results, err
}

func listTargetStreams(awsService *ec2.EC2, name string, logger log.Logger) (
	map[string]struct{}, error) {
	_, srpcClient, err := getWorkingUnpacker(awsService, name, nullLogger)
	if err != nil {
		return nil, nil
	}
	defer srpcClient.Close()
	status, err := uclient.GetStatus(srpcClient)
	if err != nil {
		logger.Println(err)
		return nil, err
	}
	streams := make(map[string]struct{})
	for stream := range status.ImageStreams {
		streams[stream] = struct{}{}
	}
	return streams, nil
}
