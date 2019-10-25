package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/Cloud-Foundations/Dominator/imagepublishers/amipublisher"
	"github.com/Cloud-Foundations/Dominator/lib/log"
)

func importKeyPairSubcommand(args []string, logger log.DebugLogger) {
	err := importKeyPair(args[0], args[1], logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error importing key pair: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func importKeyPair(keyName string, pubKeyFilename string,
	logger log.DebugLogger) error {
	publicKey, err := ioutil.ReadFile(pubKeyFilename)
	if err != nil {
		return err
	}
	return amipublisher.ImportKeyPair(targets, skipTargets, keyName, publicKey,
		logger)
}
