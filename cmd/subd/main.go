package main

import (
	"flag"
	"fmt"
	"github.com/Symantec/Dominator/lib/fsbench"
	"github.com/Symantec/Dominator/sub/fsrateio"
	"github.com/Symantec/Dominator/sub/scanner"
	"os"
	"os/exec"
	"path"
	"strconv"
	"syscall"
)

var (
	rootDir = flag.String("rootDir", "/",
		"Name of root of directory tree to manage")
	subdDir = flag.String("subdDir", "/.subd",
		"Name of subd private directory. This must be on the same file-system as rootdir")
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
		// Re-exec myself using the unshare binary as a wrapper. This hack is
		// required because syscall.Unshare() operates on only one thread in the
		// process, and Go switches execution between threads randomly. Thus,
		// the namespace can be suddenly switched for running code. This is an
		// aspect of Go that was not well thought out.
		unsharePath, err := exec.LookPath("unshare")
		if err != nil {
			fmt.Printf("Unable find unshare utility\t%s\n", err)
			return false
		}
		cmd := make([]string, 0)
		cmd = append(cmd, unsharePath)
		cmd = append(cmd, "-m")
		for _, arg := range os.Args {
			cmd = append(cmd, arg)
		}
		cmd = append(cmd, "-unshare=false")
		err = syscall.Exec(cmd[0], cmd, os.Environ())
		if err != nil {
			fmt.Printf("Unable to Exec:%s\t%s\n", cmd[0], err)
			return false
		}
	}
	// Strip out the "-unshare=false" just in case.
	os.Args = os.Args[0 : len(os.Args)-1]
	err := syscall.Mount(*rootDir, workingRootDir, "", syscall.MS_BIND, "")
	if err != nil {
		fmt.Printf("Unable to bind mount %s to %s\t%s\n",
			*rootDir, workingRootDir, err)
		return false
	}
	return true
}

func getCachedSpeed(workingRootDir string, cacheDirname string) (bytesPerSecond,
	blocksPerSecond uint64, ok bool) {
	bytesPerSecond = 0
	blocksPerSecond = 0
	devnum, err := fsbench.GetDevnumForFile(workingRootDir)
	if err != nil {
		fmt.Printf("Unable to get device number for: %s\t%s\n",
			workingRootDir, err)
		return 0, 0, false
	}
	fsbenchDir := path.Join(cacheDirname, "fsbench")
	if !createDirectory(fsbenchDir) {
		return 0, 0, false
	}
	cacheFilename := path.Join(fsbenchDir, strconv.FormatUint(devnum, 16))
	file, err := os.Open(cacheFilename)
	if err == nil {
		n, err := fmt.Fscanf(file, "%d %d", &bytesPerSecond, &blocksPerSecond)
		file.Close()
		if n == 2 || err == nil {
			return bytesPerSecond, blocksPerSecond, true
		}
	}
	bytesPerSecond, blocksPerSecond, err = fsbench.GetReadSpeed(workingRootDir)
	if err != nil {
		fmt.Printf("Unable to measure read speed\t%s\n", err)
		return 0, 0, false
	}
	file, err = os.Create(cacheFilename)
	if err != nil {
		fmt.Printf("Unable to open: %s for write\t%s\n", cacheFilename, err)
		return 0, 0, false
	}
	fmt.Fprintf(file, "%d %d\n", bytesPerSecond, blocksPerSecond)
	file.Close()
	return bytesPerSecond, blocksPerSecond, true
}

func main() {
	flag.Parse()
	workingRootDir := path.Join(*subdDir, "root")
	objectsDir := path.Join(*subdDir, "objects")
	tmpDir := path.Join(*subdDir, "tmp")
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
	bytesPerSecond, blocksPerSecond, ok := getCachedSpeed(workingRootDir,
		tmpDir)
	if !ok {
		os.Exit(1)
	}
	ctx := fsrateio.NewContext(bytesPerSecond, blocksPerSecond)
	fmt.Println(ctx)
	var prev_fs *scanner.FileSystem
	for iter := 0; true; iter++ {
		fmt.Printf("Cycle: %d\n", iter)
		fs, err := scanner.ScanFileSystem(workingRootDir, objectsDir, ctx)
		if err != nil {
			fmt.Printf("Error! %s\n", err)
			os.Exit(1)
		}
		fmt.Print(fs)
		if prev_fs != nil {
			if !scanner.Compare(prev_fs, fs, os.Stdout) {
				fmt.Println("Scan results different from last run")
			}
		}
		prev_fs = fs
	}
}
