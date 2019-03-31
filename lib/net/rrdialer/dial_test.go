package rrdialer

import (
	"context"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/log/testlogger"
)

var nextPortNumber = 12340

func (e *endpointType) makeListener(delay time.Duration,
	logger log.Logger) *uint {
	var acceptCounter uint
	e.address = "localhost:" + strconv.Itoa(nextPortNumber)
	nextPortNumber++
	go func() {
		time.Sleep(delay)
		if listener, err := net.Listen("tcp", e.address); err != nil {
			panic(err)
		} else {
			for {
				if conn, err := listener.Accept(); err != nil {
					logger.Println(err)
				} else {
					acceptCounter++
					conn.Close()
				}
			}
		}
	}()
	return &acceptCounter
}

func TestDialNoConnections(t *testing.T) {
	dialer := &Dialer{
		logger:    testlogger.New(t),
		rawDialer: &net.Dialer{Timeout: time.Second},
	}
	endpoint50 := &endpointType{
		MeanLatency: 50e-3,
	}
	endpoint100 := &endpointType{
		MeanLatency: 100e-3,
	}
	endpoints := []*endpointType{endpoint50, endpoint100}
	startTime := time.Now()
	_, err := dialer.dialEndpoints(context.Background(), "tcp",
		"localhost:1", endpoints, -1)
	if err == nil {
		t.Fatal("Dial with no working endpoints did not fail")
	}
	if time.Since(startTime) > time.Millisecond*40 {
		t.Fatal("Dial took too long to fail")
	}
}

func TestDialOneIsFastEnough(t *testing.T) {
	dialer := &Dialer{
		logger:    testlogger.New(t),
		rawDialer: &net.Dialer{Timeout: time.Second},
	}
	endpoint50 := &endpointType{
		MeanLatency: 50e-3,
	}
	endpoint100 := &endpointType{
		MeanLatency: 100e-3,
	}
	counter50 := endpoint50.makeListener(0, dialer.logger)
	counter100 := endpoint100.makeListener(time.Millisecond*40, dialer.logger)
	endpoints := []*endpointType{endpoint50, endpoint100}
	time.Sleep(time.Millisecond * 20)
	_, err := dialer.dialEndpoints(context.Background(), "tcp",
		"localhost:1", endpoints, -1)
	if err != nil {
		t.Fatal(err)
	}
	if *counter50 != 1 {
		t.Fatal("endpoint50 did not connect")
	}
	if *counter100 != 0 {
		t.Fatal("endpoint100 connected")
	}
}

func TestDialTwoAreFastEnough(t *testing.T) {
	dialer := &Dialer{
		logger:    testlogger.New(t),
		rawDialer: &net.Dialer{Timeout: time.Second},
	}
	endpoint50 := &endpointType{
		MeanLatency: 50e-3,
	}
	endpoint100 := &endpointType{
		MeanLatency: 100e-3,
	}
	endpoint150 := &endpointType{
		LastUpdate:  time.Now(),
		MeanLatency: 150e-3,
	}
	counter50 := endpoint50.makeListener(0, dialer.logger)
	counter100 := endpoint100.makeListener(time.Millisecond*40, dialer.logger)
	counter150 := endpoint100.makeListener(0, dialer.logger)
	endpoints := []*endpointType{endpoint50, endpoint100, endpoint150}
	time.Sleep(time.Millisecond * 20)
	_, err := dialer.dialEndpoints(context.Background(), "tcp",
		"localhost:1", endpoints, -1)
	if err != nil {
		t.Fatal(err)
	}
	if *counter50 != 1 && *counter150 != 1 {
		t.Fatal("endpoint50 and endpoint150 did not connect")
	}
	if *counter100 != 0 {
		t.Fatal("endpoint100 connected")
	}
}
