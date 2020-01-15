package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/sub"
	"github.com/Cloud-Foundations/Dominator/sub/client"
)

func waitForImageSubcommand(args []string, logger log.DebugLogger) error {
	srpcClient := getSubClientRetry(logger)
	defer srpcClient.Close()
	if err := waitForImage(srpcClient, args[0], logger); err != nil {
		return fmt.Errorf("Error waiting for image: %s: %s", args[0], err)
	}
	return nil
}

func waitForImage(srpcClient *srpc.Client, imageName string,
	logger log.DebugLogger) error {
	if imageName == "" {
		return errors.New("empty image name")
	}
	for time.Now().Before(timeoutTime) {
		if waitForImageLoop(srpcClient, imageName, logger) {
			return nil
		}
	}
	return errors.New("timed out waiting for update to image")
}

func waitForImageLoop(srpcClient *srpc.Client, imageName string,
	logger log.DebugLogger) bool {
	request := sub.PollRequest{ShortPollOnly: true}
	for ; time.Now().Before(timeoutTime); time.Sleep(time.Second) {
		var reply sub.PollResponse
		if err := client.CallPoll(srpcClient, request, &reply); err != nil {
			logger.Println(err)
			return false
		}
		if reply.LastSuccessfulImageName == imageName {
			return true
		}
	}
	return false
}
