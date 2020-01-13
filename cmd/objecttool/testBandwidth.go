package main

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"time"

	"github.com/Cloud-Foundations/Dominator/lib/format"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	proto "github.com/Cloud-Foundations/Dominator/proto/objectserver"
)

func testBandwidthFromServerSubcommand(args []string,
	logger log.DebugLogger) error {
	if err := testBandwidthFromServer(); err != nil {
		return fmt.Errorf("Error testing bandwidth: %s", err)
	}
	return nil
}

func testBandwidthToServerSubcommand(args []string,
	logger log.DebugLogger) error {
	if err := testBandwidthToServer(); err != nil {
		return fmt.Errorf("Error testing bandwidth: %s", err)
	}
	return nil
}

func testBandwidthFromServer() error {
	client, err := srpc.DialHTTP("tcp", fmt.Sprintf("%s:%d",
		*objectServerHostname, *objectServerPortNum), 0)
	if err != nil {
		return err
	}
	defer client.Close()
	conn, err := client.Call("ObjectServer.TestBandwidth")
	if err != nil {
		return err
	}
	defer conn.Close()
	request := proto.TestBandwidthRequest{
		Duration:  *testDuration,
		ChunkSize: *chunkSize,
	}
	if err := conn.Encode(&request); err != nil {
		return err
	}
	if err := conn.Flush(); err != nil {
		return nil
	}
	buffer := make([]byte, *chunkSize+1)
	startTime := time.Now()
	var totalBytes uint64
	for {
		if _, err := io.ReadAtLeast(conn, buffer, len(buffer)); err != nil {
			return err
		}
		totalBytes += uint64(len(buffer))
		if buffer[len(buffer)-1] == 0 {
			break
		}
	}
	localDuration := time.Since(startTime)
	var response proto.TestBandwidthResponse
	if err := conn.Decode(&response); err != nil {
		return err
	}
	localSpeed := totalBytes / uint64(localDuration.Seconds())
	serverSpeed := totalBytes / uint64(response.ServerDuration.Seconds())
	fmt.Fprintf(os.Stderr,
		"Received %s from server in %s (%s/s), at server: %s (%s/s)\n",
		format.FormatBytes(totalBytes),
		format.Duration(localDuration), format.FormatBytes(localSpeed),
		format.Duration(response.ServerDuration),
		format.FormatBytes(serverSpeed))
	return nil
}

func testBandwidthToServer() error {
	client, err := srpc.DialHTTP("tcp", fmt.Sprintf("%s:%d",
		*objectServerHostname, *objectServerPortNum), 0)
	if err != nil {
		return err
	}
	defer client.Close()
	conn, err := client.Call("ObjectServer.TestBandwidth")
	if err != nil {
		return err
	}
	defer conn.Close()
	request := proto.TestBandwidthRequest{
		Duration:     *testDuration,
		ChunkSize:    *chunkSize,
		SendToServer: true,
	}
	if err := conn.Encode(&request); err != nil {
		return err
	}
	if err := conn.Flush(); err != nil {
		return nil
	}
	buffer := make([]byte, *chunkSize+1)
	rand.Read(buffer[:request.ChunkSize])
	var totalBytes uint64
	startTime := time.Now()
	stopTime := startTime.Add(request.Duration)
	buffer[len(buffer)-1] = 1
	for time.Until(stopTime) > 0 {
		if _, err := conn.Write(buffer); err != nil {
			return err
		}
		totalBytes += uint64(len(buffer))
	}
	buffer[len(buffer)-1] = 0
	if _, err := conn.Write(buffer); err != nil {
		return err
	}
	totalBytes += uint64(len(buffer))
	if err := conn.Flush(); err != nil {
		return nil
	}
	localDuration := time.Since(startTime)
	var response proto.TestBandwidthResponse
	if err := conn.Decode(&response); err != nil {
		return err
	}
	localSpeed := totalBytes / uint64(localDuration.Seconds())
	serverSpeed := totalBytes / uint64(response.ServerDuration.Seconds())
	fmt.Fprintf(os.Stderr,
		"Sent %s to server in %s (%s/s), at server: %s (%s/s)\n",
		format.FormatBytes(totalBytes),
		format.Duration(localDuration), format.FormatBytes(localSpeed),
		format.Duration(response.ServerDuration),
		format.FormatBytes(serverSpeed))
	return nil
}
