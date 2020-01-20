package main

import (
	"fmt"
	"io/ioutil"

	"github.com/Cloud-Foundations/Dominator/imagepublishers/amipublisher"
	"github.com/Cloud-Foundations/Dominator/lib/log"
)

func importKeyPairSubcommand(args []string, logger log.DebugLogger) error {
	if err := importKeyPair(args[0], args[1], logger); err != nil {
		return fmt.Errorf("Error importing key pair: %s", err)
	}
	return nil
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
