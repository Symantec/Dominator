// +build linux

package main

import (
	"bytes"
	"fmt"
	"os"
	"syscall"

	"github.com/Symantec/Dominator/imagebuilder/builder"
	"github.com/Symantec/Dominator/lib/fsutil"
	"github.com/Symantec/Dominator/lib/log"
)

const filePerms = syscall.S_IRUSR | syscall.S_IRGRP | syscall.S_IROTH

type logWriterType struct {
	buffer bytes.Buffer
}

func buildFromManifestSubcommand(args []string, logger log.DebugLogger) {
	srpcClient := getImageServerClient()
	logWriter := &logWriterType{}
	if *alwaysShowBuildLog {
		fmt.Fprintln(os.Stderr, "Start of build log ==========================")
	}
	name, err := builder.BuildImageFromManifest(srpcClient, args[0], args[1],
		*expiresIn, logWriter, logger)
	if err != nil {
		if !*alwaysShowBuildLog {
			fmt.Fprintln(os.Stderr,
				"Start of build log ==========================")
			os.Stderr.Write(logWriter.Bytes())
		}
		fmt.Fprintln(os.Stderr, "End of build log ============================")
		fmt.Fprintf(os.Stderr, "Error processing manifest: %s\n", err)
		os.Exit(1)
	}
	if *alwaysShowBuildLog {
		fmt.Fprintln(os.Stderr, "End of build log ============================")
	} else {
		err := fsutil.CopyToFile("build.log", filePerms, &logWriter.buffer,
			uint64(logWriter.buffer.Len()))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing build log: %s\n", err)
			os.Exit(1)
		}
	}
	fmt.Println(name)
	os.Exit(0)
}

func buildTreeFromManifestSubcommand(args []string, logger log.DebugLogger) {
	srpcClient := getImageServerClient()
	logWriter := &logWriterType{}
	if *alwaysShowBuildLog {
		fmt.Fprintln(os.Stderr, "Start of build log ==========================")
	}
	rootDir, err := builder.BuildTreeFromManifest(srpcClient, args[0],
		logWriter, logger)
	if err != nil {
		if !*alwaysShowBuildLog {
			fmt.Fprintln(os.Stderr,
				"Start of build log ==========================")
			os.Stderr.Write(logWriter.Bytes())
		}
		fmt.Fprintln(os.Stderr, "End of build log ============================")
		fmt.Fprintf(os.Stderr, "Error processing manifest: %s\n", err)
		os.Exit(1)
	}
	if *alwaysShowBuildLog {
		fmt.Fprintln(os.Stderr, "End of build log ============================")
	} else {
		err := fsutil.CopyToFile("build.log", filePerms, &logWriter.buffer,
			uint64(logWriter.buffer.Len()))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing build log: %s\n", err)
			os.Exit(1)
		}
	}
	fmt.Println(rootDir)
	os.Exit(0)
}

func processManifestSubcommand(args []string, logger log.DebugLogger) {
	logWriter := &logWriterType{}
	if *alwaysShowBuildLog {
		fmt.Fprintln(os.Stderr, "Start of build log ==========================")
	}
	if err := builder.ProcessManifest(args[0], args[1], logWriter); err != nil {
		if !*alwaysShowBuildLog {
			fmt.Fprintln(os.Stderr,
				"Start of build log ==========================")
			os.Stderr.Write(logWriter.Bytes())
		}
		fmt.Fprintln(os.Stderr, "End of build log ============================")
		fmt.Fprintf(os.Stderr, "Error processing manifest: %s\n", err)
		os.Exit(1)
	}
	if *alwaysShowBuildLog {
		fmt.Fprintln(os.Stderr, "End of build log ============================")
	} else {
		err := fsutil.CopyToFile("build.log", filePerms, &logWriter.buffer,
			uint64(logWriter.buffer.Len()))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing build log: %s\n", err)
			os.Exit(1)
		}
	}
	os.Exit(0)
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
