package main

import (
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/logger"
	"os"
	"strconv"
)

func setDebugLevelSubcommand(client *srpc.Client, args []string,
	logger log.Logger) {
	level, err := strconv.ParseInt(args[0], 10, 16)
	if err != nil {
		logger.Fatalf("Error parsing level: %s\n", err)
	}
	if err := setDebugLevel(client, int16(level)); err != nil {
		logger.Fatalf("Error setting debug level: %s\n", err)
	}
	os.Exit(0)
}

func setDebugLevel(client *srpc.Client, level int16) error {
	request := proto.SetDebugLevelRequest{
		Name:  *loggerName,
		Level: level,
	}
	var reply proto.SetDebugLevelResponse
	return client.RequestReply("Logger.SetDebugLevel", request, &reply)
}
