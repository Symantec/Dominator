package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
)

func writeNetbootFilesSubcommand(args []string, logger log.DebugLogger) error {
	err := writeNetbootFiles(args[0], args[1], logger)
	if err != nil {
		return fmt.Errorf("Error writing netboot files: %s", err)
	}
	return nil
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
		false, logger)
	if err != nil {
		return err
	}
	if err := emptyTree(dirname); err != nil {
		return err
	}
	return writeConfigFiles(dirname, configFiles)
}
