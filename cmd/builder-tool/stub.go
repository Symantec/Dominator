// +build !linux

package main

import (
	"os"

	"github.com/Symantec/Dominator/lib/log"
)

func notAvailable() {
	os.Stderr.Write([]byte("Not available on this OS\n"))
	os.Exit(1)
}

func buildFromManifestSubcommand(args []string, logger log.DebugLogger) {
	notAvailable()
}

func buildRawFromManifestSubcommand(args []string, logger log.DebugLogger) {
	notAvailable()
}

func buildTreeFromManifestSubcommand(args []string, logger log.DebugLogger) {
	notAvailable()
}

func processManifestSubcommand(args []string, logger log.DebugLogger) {
	notAvailable()
}
