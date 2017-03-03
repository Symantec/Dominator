package main

import (
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/proto/sub"
	"github.com/Symantec/Dominator/sub/client"
	"log"
	"os"
	"time"
)

func waitForImageSubcommand(getSubClient getSubClientFunc, args []string) {
	if err := waitForImage(getSubClient, args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "Error waiting for image: %s: %s\n",
			args[0], err)
		os.Exit(2)
	}
	os.Exit(0)
}

func waitForImage(getSubClient getSubClientFunc, imageName string) error {
	if imageName == "" {
		return errors.New("empty image name")
	}
	logger := log.New(os.Stderr, "", log.LstdFlags)
	for time.Now().Before(timeoutTime) {
		if waitForImageLoop(getSubClient, imageName, logger) {
			return nil
		}
	}
	return errors.New("timed out waiting for update to image")
}

func waitForImageLoop(getSubClient getSubClientFunc, imageName string,
	logger *log.Logger) bool {
	srpcClient := getSubClient()
	defer srpcClient.Close()
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
