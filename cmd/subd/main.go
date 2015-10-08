package main

import (
	"flag"
	"fmt"
	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/filter"
	"github.com/Symantec/Dominator/lib/fsbench"
	"github.com/Symantec/Dominator/lib/fsrateio"
	"github.com/Symantec/Dominator/lib/html"
	"github.com/Symantec/Dominator/lib/logbuf"
	"github.com/Symantec/Dominator/lib/memstats"
	"github.com/Symantec/Dominator/lib/rateio"
	"github.com/Symantec/Dominator/sub/httpd"
	"github.com/Symantec/Dominator/sub/rpcd"
	"github.com/Symantec/Dominator/sub/scanner"
	"io"
	"log"
	"os"
	"os/signal"
	"path"
	"runtime"
	"strconv"
	"syscall"
)

var (
	logbufLines = flag.Uint("logbufLines", 1024,
		"Number of lines to store in the log buffer")
	maxThreads = flag.Uint("maxThreads", 1,
		"Maximum number of parallel OS threads to use")
	pidfile = flag.String("pidfile", "/var/run/subd.pid",
		"Name of file to write my PID to")
	portNum = flag.Uint("portNum", constants.SubPortNumber,
		"Port number to allocate and listen on for HTTP/RPC")
	rootDir = flag.String("rootDir", "/",
		"Name of root of directory tree to manage")
	showStats = flag.Bool("showStats", false,
		"If true, show statistics after each cycle")
	subdDir = flag.String("subdDir", "/.subd",
		"Name of subd private directory. This must be on the same file-system as rootDir")
	unshare = flag.Bool("unshare", true, "Internal use only.")
)

func sanityCheck() bool {
	r_devnum, err := fsbench.GetDevnumForFile(*rootDir)
	if err != nil {
		fmt.Printf("Unable to get device number for: %s\t%s\n", *rootDir, err)
		return false
	}
	s_devnum, err := fsbench.GetDevnumForFile(*subdDir)
	if err != nil {
		fmt.Printf("Unable to get device number for: %s\t%s\n", *subdDir, err)
		return false
	}
	if r_devnum != s_devnum {
		fmt.Printf("rootDir and subdDir must be on the same file-system\n")
		return false
	}
	return true
}

func createDirectory(dirname string) bool {
	err := os.MkdirAll(dirname, 0750)
	if err != nil {
		fmt.Printf("Unable to create directory: %s\t%s\n", dirname, err)
		return false
	}
	return true
}

func mountTmpfs(dirname string) bool {
	var statfs syscall.Statfs_t
	err := syscall.Statfs(dirname, &statfs)
	if err != nil {
		fmt.Printf("Unable to create Statfs: %s\t%s\n", dirname, err)
		return false
	}
	if statfs.Type != 0x01021994 {
		err := syscall.Mount("none", dirname, "tmpfs", 0,
			"size=65536,mode=0750")
		if err == nil {
			fmt.Printf("Mounted tmpfs on: %s\n", dirname)
		} else {
			fmt.Printf("Unable to mount tmpfs on: %s\t%s\n", dirname, err)
			return false
		}
	}
	return true
}

func unshareAndBind(workingRootDir string) bool {
	if *unshare {
		// Re-exec myself using the unshare syscall while on a locked thread.
		// This hack is required because syscall.Unshare() operates on only one
		// thread in the process, and Go switches execution between threads
		// randomly. Thus, the namespace can be suddenly switched for running
		// code. This is an aspect of Go that was not well thought out.
		runtime.LockOSThread()
		err := syscall.Unshare(syscall.CLONE_NEWNS)
		if err != nil {
			fmt.Printf("Unable to unshare mount namesace\t%s\n", err)
			return false
		}
		args := append(os.Args, "-unshare=false")
		err = syscall.Exec(args[0], args, os.Environ())
		if err != nil {
			fmt.Printf("Unable to Exec:%s\t%s\n", args[0], err)
			return false
		}
	}
	err := syscall.Mount("none", "/", "", syscall.MS_REC|syscall.MS_PRIVATE, "")
	if err != nil {
		fmt.Printf("Unable to set mount sharing to private\t%s\n", err)
		return false
	}
	syscall.Unmount(workingRootDir, 0)
	err = syscall.Mount(*rootDir, workingRootDir, "", syscall.MS_BIND, "")
	if err != nil {
		fmt.Printf("Unable to bind mount %s to %s\t%s\n",
			*rootDir, workingRootDir, err)
		return false
	}
	return true
}

func getCachedFsSpeed(workingRootDir string, cacheDirname string) (bytesPerSecond,
	blocksPerSecond uint64, computed, ok bool) {
	bytesPerSecond = 0
	blocksPerSecond = 0
	devnum, err := fsbench.GetDevnumForFile(workingRootDir)
	if err != nil {
		fmt.Printf("Unable to get device number for: %s\t%s\n",
			workingRootDir, err)
		return 0, 0, false, false
	}
	fsbenchDir := path.Join(cacheDirname, "fsbench")
	if !createDirectory(fsbenchDir) {
		return 0, 0, false, false
	}
	cacheFilename := path.Join(fsbenchDir, strconv.FormatUint(devnum, 16))
	file, err := os.Open(cacheFilename)
	if err == nil {
		n, err := fmt.Fscanf(file, "%d %d", &bytesPerSecond, &blocksPerSecond)
		file.Close()
		if n == 2 || err == nil {
			return bytesPerSecond, blocksPerSecond, false, true
		}
	}
	bytesPerSecond, blocksPerSecond, err = fsbench.GetReadSpeed(workingRootDir)
	if err != nil {
		fmt.Printf("Unable to measure read speed\t%s\n", err)
		return 0, 0, true, false
	}
	file, err = os.Create(cacheFilename)
	if err != nil {
		fmt.Printf("Unable to open: %s for write\t%s\n", cacheFilename, err)
		return 0, 0, true, false
	}
	fmt.Fprintf(file, "%d %d\n", bytesPerSecond, blocksPerSecond)
	file.Close()
	return bytesPerSecond, blocksPerSecond, true, true
}

func getCachedNetworkSpeed(cacheFilename string) uint64 {
	file, err := os.Open(cacheFilename)
	if err != nil {
		return 0
	}
	defer file.Close()
	var bytesPerSecond uint64
	n, err := fmt.Fscanf(file, "%d", &bytesPerSecond)
	if n == 1 || err == nil {
		return bytesPerSecond
	}
	return 0
}

type DumpableFileSystemHistory struct {
	fsh *scanner.FileSystemHistory
}

func (fsh *DumpableFileSystemHistory) WriteHtml(writer io.Writer) {
	fs := fsh.fsh.FileSystem()
	if fs == nil {
		return
	}
	fmt.Fprintln(writer, "<pre>")
	fs.List(writer)
	fmt.Fprintln(writer, "</pre>")
}

func gracefulCleanup() {
	os.Remove(*pidfile)
	os.Exit(1)
}

func writePidfile() {
	file, err := os.Create(*pidfile)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	defer file.Close()
	fmt.Fprintln(file, os.Getpid())
}

func main() {
	flag.Parse()
	workingRootDir := path.Join(*subdDir, "root")
	objectsDir := path.Join(*subdDir, "objects")
	tmpDir := path.Join(*subdDir, "tmp")
	netbenchFilename := path.Join(*subdDir, "netbench")
	oldTriggersFilename := path.Join(*subdDir, "triggers.previous")
	if !createDirectory(workingRootDir) {
		os.Exit(1)
	}
	if !sanityCheck() {
		os.Exit(1)
	}
	if !createDirectory(objectsDir) {
		os.Exit(1)
	}
	if !createDirectory(tmpDir) {
		os.Exit(1)
	}
	if !mountTmpfs(tmpDir) {
		os.Exit(1)
	}
	if !unshareAndBind(workingRootDir) {
		os.Exit(1)
	}
	runtime.GOMAXPROCS(int(*maxThreads))
	bytesPerSecond, blocksPerSecond, firstScan, ok := getCachedFsSpeed(
		workingRootDir, tmpDir)
	if !ok {
		os.Exit(1)
	}
	circularBuffer := logbuf.New(*logbufLines)
	logger := log.New(circularBuffer, "", log.LstdFlags)
	var configuration scanner.Configuration
	var err error
	configuration.ScanFilter, err = filter.NewFilter(constants.ScanExcludeList)
	if err != nil {
		fmt.Printf("Unable to set default scan exclusions\t%s\n", err)
		os.Exit(1)
	}
	configuration.FsScanContext = fsrateio.NewReaderContext(bytesPerSecond,
		blocksPerSecond, 0)
	defaultSpeed := configuration.FsScanContext.GetContext().SpeedPercent()
	if firstScan {
		configuration.FsScanContext.GetContext().SetSpeedPercent(100)
	}
	if *showStats {
		fmt.Println(configuration.FsScanContext)
	}
	var fsh scanner.FileSystemHistory
	fsChannel := scanner.StartScannerDaemon(workingRootDir, objectsDir,
		&configuration, logger)
	networkReaderContext := rateio.NewReaderContext(
		getCachedNetworkSpeed(netbenchFilename),
		constants.DefaultNetworkSpeedPercent, &rateio.ReadMeasurer{})
	configuration.NetworkReaderContext = networkReaderContext
	rescanObjectCacheChannel := rpcd.Setup(&configuration, &fsh, objectsDir,
		networkReaderContext, netbenchFilename, oldTriggersFilename, logger)
	httpd.AddHtmlWriter(&fsh)
	httpd.AddHtmlWriter(&configuration)
	httpd.AddHtmlWriter(circularBuffer)
	html.RegisterHtmlWriterForPattern("/dumpFileSystem", "Scanned File System",
		&DumpableFileSystemHistory{&fsh})
	err = httpd.StartServer(*portNum)
	if err != nil {
		fmt.Printf("Unable to create http server\t%s\n", err)
		os.Exit(1)
	}
	fsh.Update(nil)
	invalidateNextScanObjectCache := false
	sighupChannel := make(chan os.Signal)
	signal.Notify(sighupChannel, syscall.SIGHUP)
	sigtermChannel := make(chan os.Signal)
	signal.Notify(sigtermChannel, syscall.SIGTERM, syscall.SIGINT)
	writePidfile()
	for iter := 0; true; {
		select {
		case <-sighupChannel:
			err = syscall.Exec(os.Args[0], os.Args, os.Environ())
			if err != nil {
				logger.Printf("Unable to Exec:%s\t%s\n", os.Args[0], err)
			}
		case <-sigtermChannel:
			gracefulCleanup()
		case fs := <-fsChannel:
			if *showStats {
				fmt.Printf("Completed cycle: %d\n", iter)
			}
			if invalidateNextScanObjectCache {
				fs.ScanObjectCache()
				invalidateNextScanObjectCache = false
			}
			fsh.Update(fs)
			iter++
			runtime.GC() // An opportune time to take out the garbage.
			if *showStats {
				fmt.Print(fsh)
				fmt.Print(fsh.FileSystem())
				memstats.WriteMemoryStats(os.Stdout)
				fmt.Println()
			}
			if firstScan {
				configuration.FsScanContext.GetContext().SetSpeedPercent(
					defaultSpeed)
				firstScan = false
				if *showStats {
					fmt.Println(configuration.FsScanContext)
				}
			}
		case <-rescanObjectCacheChannel:
			invalidateNextScanObjectCache = true
			fsh.UpdateObjectCacheOnly()
		}
	}
}
