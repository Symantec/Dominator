package srpc

import (
	"bufio"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Cloud-Foundations/Dominator/proto/test"
)

type serverType struct{}

func init() {
	RegisterName("Test", &serverType{})
}

func makeClientServer(makeCoder coderMaker) *Client {
	serverPipe, clientPipe := net.Pipe()
	go handleConnection(&Conn{
		ReadWriter: bufio.NewReadWriter(bufio.NewReader(serverPipe),
			bufio.NewWriter(serverPipe)),
	},
		makeCoder)
	return newClient(clientPipe, clientPipe, false, makeCoder)
}

func makeListener(gob, json bool) (net.Addr, error) {
	if listener, err := net.Listen("tcp", "localhost:"); err != nil {
		return nil, err
	} else {
		serveMux := http.NewServeMux()
		if gob {
			serveMux.HandleFunc(rpcPath, gobUnsecuredHttpHandler)
		}
		if json {
			serveMux.HandleFunc(jsonRpcPath, jsonUnsecuredHttpHandler)
		}
		go func() {
			if err := http.Serve(listener, serveMux); err != nil {
				panic(err)
			}
		}()
		time.Sleep(time.Millisecond * 10) // Give the server time to start.
		return listener.Addr(), nil
	}
}

func makeListenerAndConnect(gob, json bool) (*Client, error) {
	if addr, err := makeListener(gob, json); err != nil {
		return nil, err
	} else {
		return DialHTTP(addr.Network(), addr.String(), 0)
	}
}

func testCallPlain(t *testing.T, makeCoder coderMaker) {
	client := makeClientServer(makeCoder)
	defer client.Close()
	// Call# 0.
	conn, err := client.Call("Test.Plain")
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.Encode(test.EchoRequest{Request: "plain0"}); err != nil {
		t.Fatal(err)
	}
	if err := conn.Flush(); err != nil {
		t.Fatal(err)
	}
	var response test.EchoResponse
	if err := conn.Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.Response != "plain0" {
		t.Errorf("Response: %s != plain0\n", response.Response)
	}
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}
	// Call# 1.
	conn, err = client.Call("Test.Plain")
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.Encode(test.EchoRequest{Request: "plain1"}); err != nil {
		t.Fatal(err)
	}
	if err := conn.Flush(); err != nil {
		t.Fatal(err)
	}
	if err := conn.Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.Response != "plain1" {
		t.Errorf("Response: %s != plain1\n", response.Response)
	}
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}
}

func testCallRequestReply(t *testing.T, makeCoder coderMaker) {
	serverPipe, clientPipe := net.Pipe()
	go handleConnection(&Conn{
		ReadWriter: bufio.NewReadWriter(bufio.NewReader(serverPipe),
			bufio.NewWriter(serverPipe)),
	},
		makeCoder)
	client := newClient(clientPipe, clientPipe, false, makeCoder)
	defer client.Close()
	// Call# 0.
	var response test.EchoResponse
	err := client.RequestReply("Test.RequestReply",
		test.EchoRequest{Request: "test0"}, &response)
	if err != nil {
		t.Fatal(err)
	}
	if response.Response != "test0" {
		t.Errorf("Response: %s != test0\n", response.Response)
	}
	// Call# 1.
	err = client.RequestReply("Test.RequestReply",
		test.EchoRequest{Request: "test1"}, &response)
	if err != nil {
		t.Fatal(err)
	}
	if response.Response != "test1" {
		t.Errorf("Response: %s != test1\n", response.Response)
	}
	// Call missing service.
	if _, err := client.Call("NoService.None"); err == nil {
		t.Fatal("no failure when calling unknown service")
	} else if !strings.Contains(err.Error(), "unknown service") {
		t.Fatal(err)
	}
	// Call missing method.
	if _, err := client.Call("Test.None"); err == nil {
		t.Fatal("no failure when calling unknown method")
	} else if !strings.Contains(err.Error(), "unknown method") {
		t.Fatal(err)
	}
}

func testCallReceiver(t *testing.T, makeCoder coderMaker) {
	client := makeClientServer(makeCoder)
	defer client.Close()
	// Call# 0.
	conn, err := client.Call("Test.Receiver")
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.Encode(test.EchoRequest{Request: "receiver0"}); err != nil {
		t.Fatal(err)
	}
	if err := conn.Flush(); err != nil {
		t.Fatal(err)
	}
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}
	// Call# 1. No explicit flush.
	conn, err = client.Call("Test.Receiver")
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.Encode(test.EchoRequest{Request: "receiver1"}); err != nil {
		t.Fatal(err)
	}
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}
	// Call# 2.
	conn, err = client.Call("Test.Receiver")
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.Encode(test.EchoRequest{Request: "receiver2"}); err != nil {
		t.Fatal(err)
	}
	if err := conn.Flush(); err != nil {
		t.Fatal(err)
	}
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestGobCallPlain(t *testing.T) {
	testCallPlain(t, &gobCoder{})
}

func TestJsonCallPlain(t *testing.T) {
	testCallPlain(t, &jsonCoder{})
}

func TestGobCallRequestReply(t *testing.T) {
	testCallRequestReply(t, &gobCoder{})
}

func TestJsonCallRequestReply(t *testing.T) {
	testCallRequestReply(t, &jsonCoder{})
}

func TestGobCallReceiver(t *testing.T) {
	testCallReceiver(t, &gobCoder{})
}

func TestJsonCallReceiver(t *testing.T) {
	testCallReceiver(t, &jsonCoder{})
}

func TestDualListener(t *testing.T) {
	if client, err := makeListenerAndConnect(true, true); err != nil {
		t.Fatal(err)
	} else {
		if _, ok := client.makeCoder.(*gobCoder); !ok {
			t.Fatal("GOB coder not default for dual listener")
		}
	}
}

func TestGobListener(t *testing.T) {
	if client, err := makeListenerAndConnect(true, false); err != nil {
		t.Fatal(err)
	} else {
		if _, ok := client.makeCoder.(*gobCoder); !ok {
			t.Fatal("GOB coder not available for GOB listener")
		}
	}
}

func TestJsonListener(t *testing.T) {
	if client, err := makeListenerAndConnect(false, true); err != nil {
		t.Fatal(err)
	} else {
		if _, ok := client.makeCoder.(*jsonCoder); !ok {
			t.Fatal("JSON coder not available for JSON listener")
		}
	}
}

func TestSilentListener(t *testing.T) {
	_, err := makeListenerAndConnect(false, false)
	if err != ErrorNoSrpcEndpoint {
		t.Fatalf("Silent listener error: %s != %s", err, ErrorNoSrpcEndpoint)
	}
}

func (t *serverType) Plain(conn *Conn) error {
	var request test.EchoRequest
	if err := conn.Decode(&request); err != nil {
		return err
	}
	err := conn.Encode(test.EchoResponse{Response: request.Request})
	if err != nil {
		return err
	}
	return nil
}

func (t *serverType) RequestReply(conn *Conn, request test.EchoRequest,
	response *test.EchoResponse) error {
	*response = test.EchoResponse{Response: request.Request}
	return nil
}

func (t *serverType) Receiver(conn *Conn) error {
	var request test.EchoRequest
	if err := conn.Decode(&request); err != nil {
		return err
	}
	if !strings.HasPrefix(request.Request, "receiver") {
		panic("bad request string: " + request.Request)
	}
	return nil
}
