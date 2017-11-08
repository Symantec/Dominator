package reverseconnection

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/log/prefixlogger"
	"github.com/Symantec/Dominator/lib/log/testlogger"
	libnet "github.com/Symantec/Dominator/lib/net"
)

const (
	testConnectString = "200 Connected to TestReverseDialer"
	testUrlPath       = "/_ReverseDialer_TEST_/connect"
)

var dialer *Dialer

type testAcceptEvent struct {
	conn  net.Conn
	error error
}

type serverType struct {
	logger log.DebugLogger
}

func createTestListener(logger log.DebugLogger) (*Listener, uint) {
	portNumber := 30000 + uint(rand.Intn(10000))
	listener, err := Listen("tcp", portNumber, logger)
	if err != nil {
		logger.Fatal(err)
	}
	return listener, portNumber
}

func createTestRealListener(logger log.DebugLogger) (net.Listener, uint) {
	portNumber := 30000 + uint(rand.Intn(10000))
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", portNumber))
	if err != nil {
		logger.Fatal(err)
	}
	return listener, portNumber
}

func createAcceptChannel(listener *Listener) <-chan testAcceptEvent {
	acceptChannel := make(chan testAcceptEvent, 1)
	go func() {
		conn, err := listener.Accept()
		acceptChannel <- testAcceptEvent{conn, err}
	}()
	return acceptChannel
}

func makeAndTestHttpConnection(dialer libnet.Dialer, portNumber uint,
	logger log.DebugLogger) error {
	conn, err := dialer.Dial("tcp", fmt.Sprintf("localhost:%d", portNumber))
	if err != nil {
		return errors.New("error dialing: " + err.Error())
	}
	return testHttpConnection(conn, logger)
}

func testEcho(conn net.Conn) error {
	testString := "reflect"
	writeBuffer := []byte(testString)
	if _, err := conn.Write(writeBuffer); err != nil {
		return errors.New("error writing test string: " + err.Error())
	}
	readBuffer := make([]byte, 256)
	if nRead, err := conn.Read(readBuffer); err != nil {
		return errors.New("error reading test string response: " + err.Error())
	} else if nRead != len(writeBuffer) {
		return fmt.Errorf("Wrote: %d bytes, read: %d bytes",
			len(writeBuffer), nRead)
	} else if rString := string(readBuffer[0:nRead]); rString != testString {
		return fmt.Errorf("Wrote: \"%s\", read: \"%s\"", testString, rString)
	}
	return conn.Close()
}

func testHttpConnection(conn net.Conn, logger log.DebugLogger) error {
	io.WriteString(conn, "CONNECT "+testUrlPath+" HTTP/1.0\n\n")
	resp, err := http.ReadResponse(bufio.NewReader(conn),
		&http.Request{Method: "CONNECT"})
	if err != nil {
		return errors.New("error connecting: " + err.Error())
	}
	if resp.StatusCode != http.StatusOK || resp.Status != testConnectString {
		return errors.New("unexpected HTTP response: " + resp.Status)
	}
	return testEcho(conn)
}

func (s *serverType) connectHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method != "CONNECT" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		s.logger.Println("not a hijacker ", req.RemoteAddr)
		return
	}
	conn, _, err := hijacker.Hijack()
	if err != nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		s.logger.Print("rpc hijacking ", req.RemoteAddr, ": ", err.Error())
		return
	}
	_, err = io.WriteString(conn, "HTTP/1.0 "+testConnectString+"\n\n")
	if err != nil {
		s.logger.Println("error writing connect message: ", err.Error())
		return
	}
	if _, err := io.Copy(conn, conn); err != nil {
		s.logger.Println(err)
	}
}

func TestInjectAccept(t *testing.T) {
	logger := testlogger.New(t)
	acceptChannel := make(chan acceptEvent, 1)
	fakeListener := &Listener{
		logger:        logger,
		acceptChannel: acceptChannel,
	}
	server := &serverType{logger}
	http.HandleFunc(testUrlPath, server.connectHandler)
	go http.Serve(fakeListener, nil)
	realListener, portNumber := createTestRealListener(logger)
	sc, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", portNumber))
	if err != nil {
		t.Fatal(err)
	}
	slaveConn := sc.(libnet.TCPConn)
	masterConn, err := realListener.Accept()
	if err != nil {
		t.Fatal(err)
	}
	fakeListener.acceptChannel <- acceptEvent{&Conn{TCPConn: slaveConn}, nil}
	if err := testHttpConnection(masterConn, logger); err != nil {
		t.Fatal(err)
	}
}

func TestListen(t *testing.T) {
	logger := testlogger.New(t)
	listener, portNumber := createTestListener(logger)
	acceptChannel := createAcceptChannel(listener)
	dialConn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", portNumber))
	if err != nil {
		t.Fatal(err)
	}
	response := <-acceptChannel
	if response.error != nil {
		t.Fatal(response.error)
	}
	testString := "hello"
	writeBuffer := []byte(testString)
	if _, err := dialConn.Write(writeBuffer); err != nil {
		t.Fatal(err)
	}
	readBuffer := make([]byte, 256)
	if nRead, err := response.conn.Read(readBuffer); err != nil {
		t.Fatal(err)
	} else if nRead != len(writeBuffer) {
		t.Fatalf("Wrote: %d bytes, read: %d bytes", len(writeBuffer), nRead)
	} else if rString := string(readBuffer[0:nRead]); rString != testString {
		t.Fatalf("Wrote: \"%s\", read: \"%s\"", testString, rString)
	}
	if err := dialConn.Close(); err != nil {
		t.Fatal(err)
	}
	if err := response.conn.Close(); err != nil {
		t.Fatal(err)
	}
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestListenAndHttpServe(t *testing.T) {
	logger := testlogger.New(t)
	listener, portNumber := createTestListener(logger)
	serverMux := http.NewServeMux()
	server := &serverType{logger}
	serverMux.HandleFunc(testUrlPath, server.connectHandler)
	go http.Serve(listener, serverMux)
	err := makeAndTestHttpConnection(&net.Dialer{}, portNumber, logger)
	if err != nil {
		t.Fatal(err)
	}
}

func TestReverseListenTcp(t *testing.T) {
	tLogger := testlogger.New(t)
	// Set up slave.
	slaveLogger := prefixlogger.New("slave: ", tLogger)
	slaveListener, slavePortNumber := createTestListener(slaveLogger)
	slaveAddress := fmt.Sprintf("localhost:%d", slavePortNumber)
	// Set up master
	masterLogger := prefixlogger.New("master: ", tLogger)
	masterListener, masterPortNumber := createTestRealListener(masterLogger)
	masterMux := http.NewServeMux()
	go http.Serve(masterListener, masterMux)
	dialer := NewDialer(nil, masterMux, 0, 0, masterLogger)
	// Make slave connect back to master.
	slaveLogger.Print("making slave connect to master")
	go slaveListener.connectLoop(ReverseListenerConfig{
		Network:         "tcp",
		ServerAddress:   fmt.Sprintf("127.0.0.1:%d", masterPortNumber),
		MinimumInterval: time.Millisecond,
	},
		"127.0.0.1")
	time.Sleep(time.Millisecond * 5)
	masterLogger.Print("making and testing connection")
	masterConn, err := dialer.Dial("tcp", slaveAddress)
	if err != nil {
		masterLogger.Fatal(err)
	}
	slaveConn, err := slaveListener.Accept()
	if err != nil {
		slaveLogger.Fatal(err)
	}
	if _, ok := slaveConn.(libnet.TCPConn); !ok {
		slaveLogger.Fatalf("non-TCP connection: %T", slaveConn)
	}
	go func() {
		if _, err := io.Copy(slaveConn, slaveConn); err != nil {
			slaveLogger.Println(err)
		}
	}()
	if err := testEcho(masterConn); err != nil {
		masterLogger.Fatal(err)
	}
}

func TestReverseListenHttp(t *testing.T) {
	tLogger := testlogger.New(t)
	// Set up slave.
	slaveLogger := prefixlogger.New("slave: ", tLogger)
	slaveListener, slavePortNumber := createTestListener(slaveLogger)
	slaveMux := http.NewServeMux()
	server := &serverType{slaveLogger}
	slaveMux.HandleFunc(testUrlPath, server.connectHandler)
	go http.Serve(slaveListener, slaveMux)
	// Set up master
	masterLogger := prefixlogger.New("master: ", tLogger)
	masterListener, masterPortNumber := createTestRealListener(masterLogger)
	masterMux := http.NewServeMux()
	go http.Serve(masterListener, masterMux)
	dialer := NewDialer(nil, masterMux, 0, 0, masterLogger)
	// Make slave connect back to master.
	slaveLogger.Print("making slave connect to master")
	go slaveListener.connectLoop(ReverseListenerConfig{
		Network:         "tcp",
		ServerAddress:   fmt.Sprintf("127.0.0.1:%d", masterPortNumber),
		MinimumInterval: time.Millisecond,
	},
		"127.0.0.1")
	time.Sleep(time.Millisecond * 5)
	masterLogger.Print("making and testing connection")
	err := makeAndTestHttpConnection(dialer, slavePortNumber, masterLogger)
	if err != nil {
		t.Fatal(err)
	}
}
