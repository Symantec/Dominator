package main

import (
	"os"
	"strconv"

	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	proto "github.com/Cloud-Foundations/Dominator/proto/logger"
)

func debugSubcommand(clients []*srpc.Client, addrs, args []string,
	logger log.Logger) {
	level, err := strconv.ParseUint(args[0], 10, 8)
	if err != nil {
		logger.Fatalf("Error parsing level: %s\n", err)
	}
	if err := debug(clients[0], uint8(level), args[1:]); err != nil {
		logger.Fatalf("Error sending debug log: %s\n", err)
	}
	os.Exit(0)
}

func debug(client *srpc.Client, level uint8, args []string) error {
	request := proto.DebugRequest{
		Args:  args,
		Name:  *loggerName,
		Level: level,
	}
	var reply proto.DebugResponse
	return client.RequestReply("Logger.Debug", request, &reply)
}

func printSubcommand(clients []*srpc.Client, addrs, args []string,
	logger log.Logger) {
	if err := print(clients[0], args); err != nil {
		logger.Fatalf("Error sending log: %s\n", err)
	}
	os.Exit(0)
}

func print(client *srpc.Client, args []string) error {
	request := proto.PrintRequest{
		Args: args,
		Name: *loggerName,
	}
	var reply proto.PrintResponse
	return client.RequestReply("Logger.Print", request, &reply)
}
