package main

import (
	"os"
	"strconv"

	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	proto "github.com/Cloud-Foundations/Dominator/proto/logger"
)

func setDebugLevelSubcommand(clients []*srpc.Client, addrs, args []string,
	logger log.Logger) {
	level, err := strconv.ParseInt(args[0], 10, 16)
	if err != nil {
		logger.Fatalf("Error parsing level: %s\n", err)
	}
	if err := setDebugLevel(clients[0], int16(level)); err != nil {
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
