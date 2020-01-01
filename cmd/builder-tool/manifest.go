// +build linux

package main

import (
	"bytes"
	"fmt"
	"os"
	"syscall"

	"github.com/Cloud-Foundations/Dominator/imagebuilder/builder"
	"github.com/Cloud-Foundations/Dominator/lib/fsutil"
	"github.com/Cloud-Foundations/Dominator/lib/log"
)

const filePerms = syscall.S_IRUSR | syscall.S_IRGRP | syscall.S_IROTH

type logWriterType struct {
	buffer bytes.Buffer
}

func buildFromManifestSubcommand(args []string, logger log.DebugLogger) error {
	srpcClient := getImageServerClient()
	logWriter := &logWriterType{}
	if *alwaysShowBuildLog {
		fmt.Fprintln(os.Stderr, "Start of build log ==========================")
	}
	name, err := builder.BuildImageFromManifest(srpcClient, args[0], args[1],
		*expiresIn, bindMounts, logWriter, logger)
	if err != nil {
		if !*alwaysShowBuildLog {
			fmt.Fprintln(os.Stderr,
				"Start of build log ==========================")
			os.Stderr.Write(logWriter.Bytes())
		}
		fmt.Fprintln(os.Stderr, "End of build log ============================")
		return fmt.Errorf("Error processing manifest: %s\n", err)
	}
	if *alwaysShowBuildLog {
		fmt.Fprintln(os.Stderr, "End of build log ============================")
	} else {
		err := fsutil.CopyToFile("build.log", filePerms, &logWriter.buffer,
			uint64(logWriter.buffer.Len()))
		if err != nil {
			return fmt.Errorf("Error writing build log: %s\n", err)
		}
	}
	fmt.Println(name)
	return nil
}

func buildTreeFromManifestSubcommand(args []string,
	logger log.DebugLogger) error {
	srpcClient := getImageServerClient()
	logWriter := &logWriterType{}
	if *alwaysShowBuildLog {
		fmt.Fprintln(os.Stderr, "Start of build log ==========================")
	}
	rootDir, err := builder.BuildTreeFromManifest(srpcClient, args[0],
		bindMounts, logWriter, logger)
	if err != nil {
		if !*alwaysShowBuildLog {
			fmt.Fprintln(os.Stderr,
				"Start of build log ==========================")
			os.Stderr.Write(logWriter.Bytes())
		}
		fmt.Fprintln(os.Stderr, "End of build log ============================")
		return fmt.Errorf("Error processing manifest: %s\n", err)
	}
	if *alwaysShowBuildLog {
		fmt.Fprintln(os.Stderr, "End of build log ============================")
	} else {
		err := fsutil.CopyToFile("build.log", filePerms, &logWriter.buffer,
			uint64(logWriter.buffer.Len()))
		if err != nil {
			return fmt.Errorf("Error writing build log: %s\n", err)
		}
	}
	fmt.Println(rootDir)
	return nil
}

func processManifestSubcommand(args []string, logger log.DebugLogger) error {
	logWriter := &logWriterType{}
	if *alwaysShowBuildLog {
		fmt.Fprintln(os.Stderr, "Start of build log ==========================")
	}
	err := builder.ProcessManifest(args[0], args[1], bindMounts, logWriter)
	if err != nil {
		if !*alwaysShowBuildLog {
			fmt.Fprintln(os.Stderr,
				"Start of build log ==========================")
			os.Stderr.Write(logWriter.Bytes())
		}
		fmt.Fprintln(os.Stderr, "End of build log ============================")
		return fmt.Errorf("Error processing manifest: %s\n", err)
	}
	if *alwaysShowBuildLog {
		fmt.Fprintln(os.Stderr, "End of build log ============================")
	} else {
		err := fsutil.CopyToFile("build.log", filePerms, &logWriter.buffer,
			uint64(logWriter.buffer.Len()))
		if err != nil {
			return fmt.Errorf("Error writing build log: %s\n", err)
		}
	}
	return nil
}

func (w *logWriterType) Bytes() []byte {
	return w.buffer.Bytes()
}

func (w *logWriterType) Write(p []byte) (int, error) {
	if *alwaysShowBuildLog {
		os.Stderr.Write(p)
	}
	return w.buffer.Write(p)
}
