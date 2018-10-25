package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"os"

	"github.com/Symantec/Dominator/lib/flags/loadflags"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/log/cmdlogger"
	"github.com/Symantec/Dominator/lib/srpc"
)

var (
	certFile = flag.String("certFile", "/etc/ssl/subd/cert.pem",
		"Name of file containing the SSL certificate")
	dryRun = flag.Bool("dryRun", ifUnprivileged(),
		"If true, do not make changes")
	keyFile = flag.String("keyFile", "/etc/ssl/subd/key.pem",
		"Name of file containing the SSL key")
	mountPoint = flag.String("mountPoint", "/mnt",
		"Mount point for new root file-system")
	objectsDirectory = flag.String("objectsDirectory", "/objects",
		"Directory where cached objects will be written")
	procDirectory = flag.String("procDirectory", "/proc",
		"Directory where procfs is mounted")
	sysfsDirectory = flag.String("sysfsDirectory", "/sys",
		"Directory where sysfs is mounted")
	tftpDirectory = flag.String("tftpDirectory", "/tftpdata",
		"Directory containing (possibly injected) TFTP data")
	tmpRoot = flag.String("tmpRoot", "/tmproot",
		"Mount point for temporary (tmpfs) root file-system")

	logger log.DebugLogger
)

func ifUnprivileged() bool {
	if os.Geteuid() != 0 {
		return true
	}
	return false
}

func install(logger log.DebugLogger) error {
	machineInfo, err := configureNetwork(logger)
	if err != nil {
		return err
	}
	if err := configureStorage(logger); err != nil {
		return err
	}
	_ = machineInfo
	return nil
}

func main() {
	if err := loadflags.LoadForCli("installer"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	flag.Parse()
	logger = cmdlogger.New()
	if cert, err := tls.LoadX509KeyPair(*certFile, *keyFile); err != nil {
		logger.Printf("unable to load keypair: %s", err)
	} else {
		srpc.RegisterClientTlsConfig(&tls.Config{
			InsecureSkipVerify: true,
			MinVersion:         tls.VersionTLS12,
			Certificates:       []tls.Certificate{cert},
		})
	}
	if err := install(logger); err != nil {
		logger.Fatalln(err)
	}
}
