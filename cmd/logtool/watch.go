package main

import (
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/logger"
)

func watchSubcommand(client *srpc.Client, args []string, logger log.Logger) {
	level, err := strconv.ParseInt(args[0], 10, 16)
	if err != nil {
		logger.Fatalf("Error parsing level: %s\n", err)
	}
	if err := watch(client, int16(level)); err != nil {
		logger.Fatalf("Error watching: %s\n", err)
	}
	os.Exit(0)
}

func watch(client *srpc.Client, level int16) error {
	request := proto.WatchRequest{
		ExcludeRegex: *excludeRegex,
		IncludeRegex: *includeRegex,
		Name:         *loggerName,
		DebugLevel:   level,
	}
	if conn, err := client.Call("Logger.Watch"); err != nil {
		return err
	} else {
		defer conn.Close()
		encoder := gob.NewEncoder(conn)
		if err := encoder.Encode(request); err != nil {
			return err
		}
		if err := conn.Flush(); err != nil {
			return err
		}
		decoder := gob.NewDecoder(conn)
		var response proto.WatchResponse
		if err := decoder.Decode(&response); err != nil {
			return fmt.Errorf("error decoding: %s", err)
		}
		if response.Error != "" {
			return errors.New(response.Error)
		}
		_, err := io.Copy(os.Stdout, conn)
		return err
	}
}
