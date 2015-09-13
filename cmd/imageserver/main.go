package main

import (
	"flag"
	"fmt"
	"github.com/Symantec/Dominator/imageserver/httpd"
	imageserverRpcd "github.com/Symantec/Dominator/imageserver/rpcd"
	"github.com/Symantec/Dominator/imageserver/scanner"
	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/logbuf"
	"github.com/Symantec/Dominator/objectserver/filesystem"
	objectserverRpcd "github.com/Symantec/Dominator/objectserver/rpcd"
	"log"
	"net/rpc"
	"os"
)

var (
	debug    = flag.Bool("debug", false, "If true, show debugging output")
	imageDir = flag.String("imageDir", "/var/lib/imageserver",
		"Name of image server data directory.")
	logbufLines = flag.Uint("logbufLines", 1024,
		"Number of lines to store in the log buffer")
	objectDir = flag.String("objectDir", "/var/lib/objectserver",
		"Name of image server data directory.")
	portNum = flag.Uint("portNum", constants.ImageServerPortNumber,
		"Port number to allocate and listen on for HTTP/RPC")
)

func main() {
	flag.Parse()
	if os.Geteuid() == 0 {
		fmt.Println("Do not run the Image Server as root")
		os.Exit(1)
	}
	circularBuffer := logbuf.New(*logbufLines)
	logger := log.New(circularBuffer, "", log.LstdFlags)
	objSrv, err := filesystem.NewObjectServer(*objectDir, logger)
	if err != nil {
		fmt.Printf("Cannot create ObjectServer\t%s\n", err)
		os.Exit(1)
	}
	imdb, err := scanner.LoadImageDataBase(*imageDir, objSrv)
	if err != nil {
		fmt.Printf("Cannot load image database\t%s\n", err)
		os.Exit(1)
	}
	imageserverRpcd.Setup(imdb, logger)
	rpcHtmlWriter := objectserverRpcd.Setup(objSrv, logger)
	rpc.HandleHTTP()
	httpd.AddHtmlWriter(imdb)
	httpd.AddHtmlWriter(rpcHtmlWriter)
	httpd.AddHtmlWriter(circularBuffer)
	err = httpd.StartServer(*portNum, imdb, false)
	if err != nil {
		fmt.Printf("Unable to create http server\t%s\n", err)
		os.Exit(1)
	}
}
