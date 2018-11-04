package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
)

func writeNetbootFilesSubcommand(args []string, logger log.DebugLogger) {
	err := writeNetbootFiles(args[0], args[1], logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing netboot files: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func emptyTree(rootDir string) error {
	dir, err := os.Open(rootDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	names, err := dir.Readdirnames(-1)
	dir.Close()
	if err != nil {
		return err
	}
	for _, name := range names {
		if err := os.Remove(filepath.Join(rootDir, name)); err != nil {
			return err
		}
	}
	return nil
}

func writeNetbootFiles(hostname, dirname string, logger log.DebugLogger) error {
	fmCR := srpc.NewClientResource("tcp",
		fmt.Sprintf("%s:%d", *fleetManagerHostname, *fleetManagerPortNum))
	defer fmCR.ScheduleClose()
	imageClient, err := srpc.DialHTTP("tcp", fmt.Sprintf("%s:%d",
		*imageServerHostname, *imageServerPortNum), 0)
	if err != nil {
		return err
	}
	defer imageClient.Close()
	_, _, configFiles, err := getInstallConfig(fmCR, imageClient, hostname,
		logger)
	if err != nil {
		return err
	}
	if err := emptyTree(dirname); err != nil {
		return err
	}
	return writeConfigFiles(dirname, configFiles)
}
