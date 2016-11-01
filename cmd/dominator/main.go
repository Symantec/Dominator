package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/Symantec/Dominator/dom/herd"
	"github.com/Symantec/Dominator/dom/rpcd"
	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/logbuf"
	"github.com/Symantec/Dominator/lib/mdb"
	"github.com/Symantec/Dominator/lib/mdb/mdbd"
	objectserver "github.com/Symantec/Dominator/lib/objectserver/filesystem"
	"github.com/Symantec/Dominator/lib/srpc/setupserver"
	"github.com/Symantec/Dominator/lib/wsyscall"
	"github.com/Symantec/tricorder/go/tricorder"
	"log"
	"os"
	"os/user"
	"path"
	"runtime"
	"strconv"
	"syscall"
	"time"
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
	mdbFile = flag.String("mdbFile", "mdb",
		"File to read MDB data from, relative to stateDir (default format is JSON)")
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
	username = flag.String("username", "",
		"If running as root, username to switch to.")
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

func setUser(username string) error {
	// Lock to OS thread so that UID change sticks to this goroutine and the
	// re-exec at the end. wsyscall.SetAllUid() only affects one thread on
	// Linux.
	runtime.LockOSThread()
	if username == "" {
		return errors.New("-username argument missing")
	}
	newUser, err := user.Lookup(username)
	if err != nil {
		return err
	}
	uid, err := strconv.Atoi(newUser.Uid)
	if err != nil {
		return err
	}
	gid, err := strconv.Atoi(newUser.Gid)
	if err != nil {
		return err
	}
	if uid == 0 {
		return errors.New("Do not run the Dominator as root")
		os.Exit(1)
	}
	if err := wsyscall.SetAllGid(gid); err != nil {
		return err
	}
	if err := wsyscall.SetAllUid(uid); err != nil {
		return err
	}
	return syscall.Exec(os.Args[0], os.Args, os.Environ())
}

func pathJoin(first, second string) string {
	if path.IsAbs(second) {
		return path.Clean(second)
	}
	return path.Join(first, second)
}

func newObjectServer(objectsDir string, logger *log.Logger) (
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
	flag.Parse()
	tricorder.RegisterFlags()
	circularBuffer := logbuf.New()
	logger := log.New(circularBuffer, "", log.LstdFlags)
	if err := setupserver.SetupTls(); err != nil {
		logger.Println(err)
		circularBuffer.Flush()
		if !*permitInsecureMode {
			os.Exit(1)
		}
	}
	rlim := syscall.Rlimit{*fdLimit, *fdLimit}
	if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rlim); err != nil {
		fmt.Fprintf(os.Stderr, "Cannot set FD limit\t%s\n", err)
		os.Exit(1)
	}
	if os.Geteuid() == 0 {
		if err := setUser(*username); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
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
	mdbChannel := mdbd.StartMdbDaemon(path.Join(*stateDir, *mdbFile), logger)
	objectServer, err := newObjectServer(path.Join(*stateDir, *objectsDir),
		logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot load objectcache: %s\n", err)
		os.Exit(1)
	}
	herd := herd.NewHerd(fmt.Sprintf("%s:%d", *imageServerHostname,
		*imageServerPortNum), objectServer, logger)
	herd.AddHtmlWriter(circularBuffer)
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
