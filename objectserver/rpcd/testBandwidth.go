package rpcd

import (
	"fmt"
	"io"
	"math/rand"
	"time"

	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	proto "github.com/Cloud-Foundations/Dominator/proto/objectserver"
)

func (t *srpcType) TestBandwidth(conn *srpc.Conn) error {
	var request proto.TestBandwidthRequest
	if err := conn.Decode(&request); err != nil {
		return err
	}
	if request.ChunkSize > 65535 {
		return fmt.Errorf("ChunkSize: %d exceeds 65535", request.ChunkSize)
	}
	if request.SendToServer {
		return t.testBandwidthToServer(conn, request)
	}
	return t.testBandwidthToClient(conn, request)
}

func (t *srpcType) testBandwidthToClient(conn *srpc.Conn,
	request proto.TestBandwidthRequest) error {
	buffer := make([]byte, request.ChunkSize+1)
	rand.Read(buffer[:request.ChunkSize])
	if request.Duration < time.Second {
		request.Duration = time.Second
	} else if request.Duration > time.Minute {
		request.Duration = time.Minute
	}
	startTime := time.Now()
	stopTime := startTime.Add(request.Duration)
	buffer[len(buffer)-1] = 1
	for time.Until(stopTime) > 0 {
		if _, err := conn.Write(buffer); err != nil {
			return err
		}
	}
	buffer[len(buffer)-1] = 0
	if _, err := conn.Write(buffer); err != nil {
		return err
	}
	return conn.Encode(&proto.TestBandwidthResponse{time.Since(startTime)})
}

func (t *srpcType) testBandwidthToServer(conn *srpc.Conn,
	request proto.TestBandwidthRequest) error {
	buffer := make([]byte, request.ChunkSize+1)
	startTime := time.Now()
	for {
		if _, err := io.ReadAtLeast(conn, buffer, len(buffer)); err != nil {
			return err
		}
		if buffer[len(buffer)-1] == 0 {
			break
		}
	}
	return conn.Encode(&proto.TestBandwidthResponse{time.Since(startTime)})
}
