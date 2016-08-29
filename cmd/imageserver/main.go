package main

import (
	"flag"
	"fmt"
	"github.com/Symantec/Dominator/imageserver/httpd"
	imageserverRpcd "github.com/Symantec/Dominator/imageserver/rpcd"
	"github.com/Symantec/Dominator/imageserver/scanner"
	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/logbuf"
	"github.com/Symantec/Dominator/lib/objectserver/filesystem"
	"github.com/Symantec/Dominator/lib/srpc/setupserver"
	objectserverRpcd "github.com/Symantec/Dominator/objectserver/rpcd"
	"github.com/Symantec/tricorder/go/tricorder"
	"github.com/Symantec/tricorder/go/tricorder/units"
	"log"
	"os"
)

var (
	archiveExpiringImages = flag.Bool("archiveExpiringImages", false,
		"If true, replicate expiring images when in archive mode")
	archiveMode = flag.Bool("archiveMode", false,
		"If true, disable delete operations and require update server")
	debug    = flag.Bool("debug", false, "If true, show debugging output")
	imageDir = flag.String("imageDir", "/var/lib/imageserver",
		"Name of image server data directory.")
	imageServerHostname = flag.String("imageServerHostname", "",
		"Hostname of image server to receive updates from")
	imageServerPortNum = flag.Uint("imageServerPortNum",
		constants.ImageServerPortNumber,
		"Port number of image server")
	logbufLines = flag.Uint("logbufLines", 1024,
		"Number of lines to store in the log buffer")
	objectDir = flag.String("objectDir", "/var/lib/objectserver",
		"Name of image server data directory.")
	permitInsecureMode = flag.Bool("permitInsecureMode", false,
		"If true, run in insecure mode. This gives remote access to all")
	portNum = flag.Uint("portNum", constants.ImageServerPortNumber,
		"Port number to allocate and listen on for HTTP/RPC")
)

type imageObjectServersType struct {
	imdb   *scanner.ImageDataBase
	objSrv *filesystem.ObjectServer
}

func main() {
	flag.Parse()
	tricorder.RegisterFlags()
	if os.Geteuid() == 0 {
		fmt.Fprintln(os.Stderr, "Do not run the Image Server as root")
		os.Exit(1)
	}
	if *archiveMode && *imageServerHostname == "" {
		fmt.Fprintln(os.Stderr, "-imageServerHostname required in archive mode")
		os.Exit(1)
	}
	circularBuffer := logbuf.New(*logbufLines)
	logger := log.New(circularBuffer, "", log.LstdFlags)
	if err := setupserver.SetupTls(); err != nil {
		logger.Println(err)
		circularBuffer.Flush()
		if !*permitInsecureMode {
			os.Exit(1)
		}
	}
	objSrv, err := filesystem.NewObjectServer(*objectDir, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot create ObjectServer: %s\n", err)
		os.Exit(1)
	}
	imdb, err := scanner.LoadImageDataBase(*imageDir, objSrv, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot load image database: %s\n", err)
		os.Exit(1)
	}
	tricorder.RegisterMetric("/image-count",
		func() uint { return imdb.CountImages() },
		units.None, "number of images")
	imgSrvRpcHtmlWriter := imageserverRpcd.Setup(imdb, *imageServerHostname,
		logger)
	objSrvRpcHtmlWriter := objectserverRpcd.Setup(objSrv, logger)
	httpd.AddHtmlWriter(imdb)
	httpd.AddHtmlWriter(&imageObjectServersType{imdb, objSrv})
	httpd.AddHtmlWriter(imgSrvRpcHtmlWriter)
	httpd.AddHtmlWriter(objSrvRpcHtmlWriter)
	httpd.AddHtmlWriter(circularBuffer)
	if *imageServerHostname != "" {
		go replicator(fmt.Sprintf("%s:%d", *imageServerHostname,
			*imageServerPortNum), imdb, objSrv, *archiveMode, logger)
	}
	if err = httpd.StartServer(*portNum, imdb, objSrv, false); err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create http server: %s\n", err)
		os.Exit(1)
	}
}
