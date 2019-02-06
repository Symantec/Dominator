// +build linux

package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"os"

	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/imaginator"
)

func buildImageSubcommand(args []string, logger log.Logger) {
	if err := buildImage(args, logger); err != nil {
		fmt.Fprintf(os.Stderr, "Error building image: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func buildImage(args []string, logger log.Logger) error {
	srpcClient := getImaginatorClient()
	request := proto.BuildImageRequest{
		StreamName:     args[0],
		ExpiresIn:      *expiresIn,
		MaxSourceAge:   *maxSourceAge,
		StreamBuildLog: true,
	}
	if len(args) > 1 {
		request.GitBranch = args[1]
	}
	logBuffer := &bytes.Buffer{}
	var logWriter io.Writer
	if *alwaysShowBuildLog {
		fmt.Fprintln(os.Stderr, "Start of build log ==========================")
		logWriter = os.Stderr
	} else {
		logWriter = logBuffer
	}
	var reply proto.BuildImageResponse
	err := callBuildImage(srpcClient, request, &reply, logWriter)
	if err != nil {
		if !*alwaysShowBuildLog {
			os.Stderr.Write(logBuffer.Bytes())
		}
		fmt.Fprintln(os.Stderr, "End of build log ============================")
		return err
	}
	if *alwaysShowBuildLog {
		fmt.Fprintln(os.Stderr, "End of build log ============================")
	}
	fmt.Println(reply.ImageName)
	return nil
}

func callBuildImage(client *srpc.Client, request proto.BuildImageRequest,
	response *proto.BuildImageResponse, logWriter io.Writer) error {
	conn, err := client.Call("Imaginator.BuildImage")
	if err != nil {
		return err
	}
	defer conn.Close()
	decoder := gob.NewDecoder(conn)
	encoder := gob.NewEncoder(conn)
	if err := encoder.Encode(request); err != nil {
		return err
	}
	if err := conn.Flush(); err != nil {
		return err
	}
	str, err := conn.ReadString('\n')
	if err != nil {
		return err
	}
	if str != "\n" {
		return errors.New(str[:len(str)-1])
	}
	for {
		var reply proto.BuildImageResponse
		if err := decoder.Decode(&reply); err != nil {
			return fmt.Errorf("error reading reply: %s", err)
		}
		if err := errors.New(reply.ErrorString); err != nil {
			return err
		}
		logWriter.Write(reply.BuildLog)
		reply.BuildLog = nil
		if reply.ImageName != "" {
			*response = reply
			return nil
		}
	}
}
