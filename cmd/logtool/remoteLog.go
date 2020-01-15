package main

import (
	"fmt"
	"strconv"

	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	proto "github.com/Cloud-Foundations/Dominator/proto/logger"
)

func debugSubcommand(args []string, logger log.DebugLogger) error {
	level, err := strconv.ParseUint(args[0], 10, 8)
	if err != nil {
		return fmt.Errorf("Error parsing level: %s", err)
	}
	clients, _, err := dial(false)
	if err != nil {
		return err
	}
	if err := debug(clients[0], uint8(level), args[1:]); err != nil {
		logger.Fatalf("Error sending debug log: %s\n", err)
	}
	return nil
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

func printSubcommand(args []string, logger log.DebugLogger) error {
	clients, _, err := dial(false)
	if err != nil {
		return err
	}
	if err := print(clients[0], args); err != nil {
		logger.Fatalf("Error sending log: %s\n", err)
	}
	return nil
}

func print(client *srpc.Client, args []string) error {
	request := proto.PrintRequest{
		Args: args,
		Name: *loggerName,
	}
	var reply proto.PrintResponse
	return client.RequestReply("Logger.Print", request, &reply)
}
