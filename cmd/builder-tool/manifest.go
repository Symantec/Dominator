package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"syscall"

	"github.com/Symantec/Dominator/imagebuilder/builder"
	"github.com/Symantec/Dominator/lib/fsutil"
	"github.com/Symantec/Dominator/lib/log"
)

const filePerms = syscall.S_IRUSR | syscall.S_IRGRP | syscall.S_IROTH

func buildFromManifestSubcommand(args []string, logger log.Logger) {
	srpcClient := getImageServerClient()
	buildLog := &bytes.Buffer{}
	name, err := builder.BuildImageFromManifest(srpcClient, args[0], args[1],
		*expiresIn, buildLog, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error processing manifest: %s\n", err)
		io.Copy(os.Stderr, buildLog)
		os.Exit(1)
	}
	fmt.Println(name)
	os.Exit(0)
}

func buildTreeFromManifestSubcommand(args []string, logger log.Logger) {
	srpcClient := getImageServerClient()
	buildLog := &bytes.Buffer{}
	rootDir, err := builder.BuildTreeFromManifest(srpcClient, args[0], buildLog,
		logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error processing manifest: %s\n", err)
		io.Copy(os.Stderr, buildLog)
		os.Exit(1)
	}
	fmt.Println(rootDir)
	err = fsutil.CopyToFile("build.log", filePerms, buildLog,
		uint64(buildLog.Len()))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing build log: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func processManifestSubcommand(args []string, logger log.Logger) {
	buildLog := &bytes.Buffer{}
	if err := builder.ProcessManifest(args[0], args[1], buildLog); err != nil {
		fmt.Fprintf(os.Stderr, "Error processing manifest: %s\n", err)
		io.Copy(os.Stderr, buildLog)
		os.Exit(1)
	} else {
		io.Copy(os.Stdout, buildLog)
	}
	os.Exit(0)
}
