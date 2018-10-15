package main

import (
	"encoding/gob"
	"fmt"
	"io"
	"math/rand"
	"os"
	"time"

	"github.com/Symantec/Dominator/lib/format"
	"github.com/Symantec/Dominator/lib/objectserver"
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/objectserver"
)

func testBandwidthFromServerSubcommand(objSrv objectserver.ObjectServer,
	args []string) {
	if err := testBandwidthFromServer(); err != nil {
		fmt.Fprintf(os.Stderr, "Error testing bandwidth: %s\n", err)
		os.Exit(2)
	}
	os.Exit(0)
}

func testBandwidthToServerSubcommand(objSrv objectserver.ObjectServer,
	args []string) {
	if err := testBandwidthToServer(); err != nil {
		fmt.Fprintf(os.Stderr, "Error testing bandwidth: %s\n", err)
		os.Exit(2)
	}
	os.Exit(0)
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
	decoder := gob.NewDecoder(conn)
	encoder := gob.NewEncoder(conn)
	request := proto.TestBandwidthRequest{
		Duration:  *testDuration,
		ChunkSize: *chunkSize,
	}
	if err := encoder.Encode(&request); err != nil {
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
	if err := decoder.Decode(&response); err != nil {
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
	decoder := gob.NewDecoder(conn)
	encoder := gob.NewEncoder(conn)
	request := proto.TestBandwidthRequest{
		Duration:     *testDuration,
		ChunkSize:    *chunkSize,
		SendToServer: true,
	}
	if err := encoder.Encode(&request); err != nil {
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
	if err := decoder.Decode(&response); err != nil {
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
