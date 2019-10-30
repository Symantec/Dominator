package amipublisher

import (
	"github.com/Cloud-Foundations/Dominator/lib/awsutil"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func importKeyPair(targets awsutil.TargetList, skipList awsutil.TargetList,
	keyName string, publicKey []byte, logger log.Logger) error {
	resultsChannel := make(chan error, 1)
	numTargets, err := awsutil.ForEachTarget(targets, skipList,
		func(awsService *ec2.EC2, account, region string, logger log.Logger) {
			err := importKeyPairInTarget(awsService, keyName, publicKey, logger)
			if err != nil {
				logger.Println(err)
			}
			resultsChannel <- err
		},
		logger)
	// Collect results.
	for i := 0; i < numTargets; i++ {
		e := <-resultsChannel
		if e != nil && err == nil {
			err = e
		}
	}
	return err
}

func importKeyPairInTarget(awsService *ec2.EC2, keyName string,
	publicKey []byte, logger log.Logger) error {
	out, err := awsService.DescribeKeyPairs(&ec2.DescribeKeyPairsInput{
		KeyNames: aws.StringSlice([]string{keyName}),
	})
	if err == nil && len(out.KeyPairs) == 1 {
		return nil
	}
	_, err = awsService.ImportKeyPair(&ec2.ImportKeyPairInput{
		KeyName:           aws.String(keyName),
		PublicKeyMaterial: publicKey,
	})
	return err
}
