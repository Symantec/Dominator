package main

import (
	"fmt"
	"strconv"

	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	proto "github.com/Cloud-Foundations/Dominator/proto/logger"
)

func setDebugLevelSubcommand(args []string, logger log.DebugLogger) error {
	level, err := strconv.ParseInt(args[0], 10, 16)
	if err != nil {
		return fmt.Errorf("Error parsing level: %s", err)
	}
	clients, _, err := dial(false)
	if err != nil {
		return err
	}
	if err := setDebugLevel(clients[0], int16(level)); err != nil {
		return fmt.Errorf("Error setting debug level: %s", err)
	}
	return nil
}

func setDebugLevel(client *srpc.Client, level int16) error {
	request := proto.SetDebugLevelRequest{
		Name:  *loggerName,
		Level: level,
	}
	var reply proto.SetDebugLevelResponse
	return client.RequestReply("Logger.SetDebugLevel", request, &reply)
}
