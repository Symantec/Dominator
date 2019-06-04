package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/Symantec/Dominator/lib/flags/loadflags"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/log/serverlogger"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/lib/srpc/setupserver"
	"github.com/Symantec/Dominator/proto/test"
	"github.com/Symantec/tricorder/go/tricorder"
)

var (
	permitInsecureMode = flag.Bool("permitInsecureMode", false,
		"If true, run in insecure mode. This gives remote access to all")
	portNum = flag.Uint("portNum", 12345,
		"Port number to allocate and listen on for HTTP/RPC")
)

type serverType struct{}

func doMain(logger log.DebugLogger) error {
	if err := setupserver.SetupTls(); err != nil {
		if *permitInsecureMode {
			logger.Println(err)
		} else {
			return err
		}
	}
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", *portNum))
	if err != nil {
		return err
	}
	srpc.RegisterName("Test", &serverType{})
	return http.Serve(listener, nil)
}

func main() {
	if err := loadflags.LoadForDaemon("srpc-test"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	flag.Parse()
	tricorder.RegisterFlags()
	logger := serverlogger.New("")
	if err := doMain(logger); err != nil {
		logger.Fatalln(err)
	}
}

func (t *serverType) RequestReply(conn *srpc.Conn, request test.EchoRequest,
	response *test.EchoResponse) error {
	*response = test.EchoResponse{Response: request.Request}
	return nil
}
