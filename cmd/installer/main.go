// +build linux

package main

import (
	"flag"
	"fmt"
	stdlog "log"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/flags/loadflags"
	"github.com/Symantec/Dominator/lib/fsutil"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/log/debuglogger"
	"github.com/Symantec/Dominator/lib/logbuf"
	"github.com/Symantec/Dominator/lib/srpc/setupserver"
	"github.com/Symantec/tricorder/go/tricorder"
)

const logfile = "/var/log/installer/latest"

var (
	dryRun = flag.Bool("dryRun", ifUnprivileged(),
		"If true, do not make changes")
	mountPoint = flag.String("mountPoint", "/mnt",
		"Mount point for new root file-system")
	objectsDirectory = flag.String("objectsDirectory", "/objects",
		"Directory where cached objects will be written")
	logDebugLevel = flag.Int("logDebugLevel", -1, "Debug log level")
	portNum       = flag.Uint("portNum", constants.InstallerPortNumber,
		"Port number to allocate and listen on for HTTP/RPC")
	procDirectory = flag.String("procDirectory", "/proc",
		"Directory where procfs is mounted")
	skipNetwork = flag.Bool("skipNetwork", false,
		"If true, do not update target network configuration")
	skipStorage = flag.Bool("skipStorage", false,
		"If true, do not update storage")
	sysfsDirectory = flag.String("sysfsDirectory", "/sys",
		"Directory where sysfs is mounted")
	tftpDirectory = flag.String("tftpDirectory", "/tftpdata",
		"Directory containing (possibly injected) TFTP data")
	tmpRoot = flag.String("tmpRoot", "/tmproot",
		"Mount point for temporary (tmpfs) root file-system")
)

func copyLogs() error {
	logdir := filepath.Join(*mountPoint, "var", "log", "installer")
	return fsutil.CopyFile(filepath.Join(logdir, "log"), logfile, filePerms)
}

func createLogger() (*logbuf.LogBuffer, log.DebugLogger) {
	os.MkdirAll("/var/log/installer", dirPerms)
	options := logbuf.GetStandardOptions()
	options.AlsoLogToStderr = true
	logBuffer := logbuf.NewWithOptions(options)
	logger := debuglogger.New(stdlog.New(logBuffer, "", 0))
	logger.SetLevel(int16(*logDebugLevel))
	return logBuffer, logger
}

func ifUnprivileged() bool {
	if os.Geteuid() != 0 {
		return true
	}
	return false
}

func install(logger log.DebugLogger) error {
	machineInfo, interfaces, err := configureLocalNetwork(logger)
	if err != nil {
		return err
	}
	if !*skipStorage {
		if err := configureStorage(*machineInfo, logger); err != nil {
			return err
		}
	}
	if !*skipNetwork {
		err := configureNetwork(*machineInfo, interfaces, logger)
		if err != nil {
			return err
		}
	}
	if err := copyLogs(); err != nil {
		return fmt.Errorf("error copying logs: %s", err)
	}
	syscall.Sync()
	time.Sleep(time.Second)
	if err := unmountStorage(logger); err != nil {
		return fmt.Errorf("error unmounting: %s", err)
	}
	return nil
}

func main() {
	if err := loadflags.LoadForDaemon("installer"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	flag.Parse()
	tricorder.RegisterFlags()
	logBuffer, logger := createLogger()
	defer logBuffer.Flush()
	go runShellOnConsole(logger)
	AddHtmlWriter(logBuffer)
	if err := setupserver.SetupTls(); err != nil {
		logger.Println(err)
	}
	if newLogger, err := startServer(*portNum, logger); err != nil {
		logger.Printf("cannot start server: %s\n", err)
	} else {
		logger = newLogger
	}
	if err := install(logger); err != nil {
		logger.Println(err)
		logger.Println("waiting 5m before rebooting")
		time.Sleep(time.Minute * 5)
	} else {
		logger.Println("waiting 5s before rebooting")
		time.Sleep(time.Second * 5)
	}
	syscall.Sync()
	if err := syscall.Reboot(syscall.LINUX_REBOOT_CMD_RESTART); err != nil {
		logger.Fatalf("error rebooting: %s\n", err)
	}
}
