package main

import (
	"flag"
	"fmt"
	_ "net/http/pprof"
	"os"
	"path"
	"syscall"
	"time"

	"github.com/Cloud-Foundations/Dominator/dom/herd"
	"github.com/Cloud-Foundations/Dominator/dom/rpcd"
	"github.com/Cloud-Foundations/Dominator/lib/constants"
	"github.com/Cloud-Foundations/Dominator/lib/flags/loadflags"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/log/serverlogger"
	"github.com/Cloud-Foundations/Dominator/lib/mdb"
	"github.com/Cloud-Foundations/Dominator/lib/mdb/mdbd"
	objectserver "github.com/Cloud-Foundations/Dominator/lib/objectserver/filesystem"
	"github.com/Cloud-Foundations/Dominator/lib/srpc/setupserver"
	"github.com/Cloud-Foundations/tricorder/go/tricorder"
)

const dirPerms = syscall.S_IRWXU

var (
	debug = flag.Bool("debug", false,
		"If true, show debugging output")
	fdLimit = flag.Uint64("fdLimit", getFdLimit(),
		"Maximum number of open file descriptors (this limits concurrent connection attempts)")
	imageServerHostname = flag.String("imageServerHostname", "localhost",
		"Hostname of image server")
	imageServerPortNum = flag.Uint("imageServerPortNum",
		constants.ImageServerPortNumber,
		"Port number of image server")
	mdbFile = flag.String("mdbFile", constants.DefaultMdbFile,
		"File to read MDB data from")
	minInterval = flag.Uint("minInterval", 1,
		"Minimum interval between loops (in seconds)")
	objectsDir = flag.String("objectsDir", "objects",
		"Directory containing computed objects, relative to stateDir")
	permitInsecureMode = flag.Bool("permitInsecureMode", false,
		"If true, run in insecure mode. This gives remote access to all")
	portNum = flag.Uint("portNum", constants.DominatorPortNumber,
		"Port number to allocate and listen on for HTTP/RPC")
	stateDir = flag.String("stateDir", "/var/lib/Dominator",
		"Name of dominator state directory.")
)

func showMdb(mdb *mdb.Mdb) {
	fmt.Println()
	mdb.DebugWrite(os.Stdout)
	fmt.Println()
}

func getFdLimit() uint64 {
	var rlim syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rlim); err != nil {
		panic(err)
	}
	return rlim.Max
}

func pathJoin(first, second string) string {
	if path.IsAbs(second) {
		return path.Clean(second)
	}
	return path.Join(first, second)
}

func newObjectServer(objectsDir string, logger log.DebugLogger) (
	*objectserver.ObjectServer, error) {
	fi, err := os.Stat(objectsDir)
	if err != nil {
		if err := os.Mkdir(objectsDir, dirPerms); err != nil {
			return nil, err
		}
	} else if !fi.IsDir() {
		return nil, fmt.Errorf("%s is not a directory\n", objectsDir)
	}
	return objectserver.NewObjectServer(objectsDir, logger)
}

func main() {
	if os.Geteuid() == 0 {
		fmt.Fprintln(os.Stderr, "Do not run the Dominator as root")
		os.Exit(1)
	}
	if err := loadflags.LoadForDaemon("dominator"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	flag.Parse()
	tricorder.RegisterFlags()
	logger := serverlogger.New("")
	if err := setupserver.SetupTls(); err != nil {
		if *permitInsecureMode {
			logger.Println(err)
		} else {
			logger.Fatalln(err)
		}
	}
	rlim := syscall.Rlimit{Cur: *fdLimit, Max: *fdLimit}
	if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rlim); err != nil {
		fmt.Fprintf(os.Stderr, "Cannot set FD limit\t%s\n", err)
		os.Exit(1)
	}
	fi, err := os.Lstat(*stateDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot stat: %s\t%s\n", *stateDir, err)
		os.Exit(1)
	}
	if !fi.IsDir() {
		fmt.Fprintf(os.Stderr, "%s is not a directory\n", *stateDir)
		os.Exit(1)
	}
	interval := time.Duration(*minInterval) * time.Second
	mdbChannel := mdbd.StartMdbDaemon(*mdbFile, logger)
	objectServer, err := newObjectServer(path.Join(*stateDir, *objectsDir),
		logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot load objectcache: %s\n", err)
		os.Exit(1)
	}
	metricsDir, err := tricorder.RegisterDirectory("/dominator/herd")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot create metrics directory: %s\n", err)
		os.Exit(1)
	}
	herd := herd.NewHerd(fmt.Sprintf("%s:%d", *imageServerHostname,
		*imageServerPortNum), objectServer, metricsDir, logger)
	herd.AddHtmlWriter(logger)
	rpcd.Setup(herd, logger)
	if err = herd.StartServer(*portNum, true); err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create http server\t%s\n", err)
		os.Exit(1)
	}
	scanTokenChannel := make(chan bool, 1)
	scanTokenChannel <- true
	nextCycleStopTime := time.Now().Add(interval)
	for {
		select {
		case mdb := <-mdbChannel:
			herd.MdbUpdate(mdb)
			if *debug {
				showMdb(mdb)
			}
		case <-scanTokenChannel:
			// Scan one sub.
			if herd.PollNextSub() { // We've reached the end of a scan cycle.
				if *debug {
					fmt.Print(".")
				}
				go func(sleepDuration time.Duration) {
					time.Sleep(sleepDuration)
					nextCycleStopTime = time.Now().Add(interval)
					scanTokenChannel <- true
				}(nextCycleStopTime.Sub(time.Now()))
			} else {
				scanTokenChannel <- true
			}
		}
	}
}
