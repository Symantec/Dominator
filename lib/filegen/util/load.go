package util

import (
	"fmt"
	"github.com/Symantec/Dominator/lib/filegen"
	"github.com/Symantec/Dominator/lib/fsutil"
	"strings"
)

type configFunc func(*filegen.Manager, []string) error

type configType struct {
	minFields  int
	maxFields  int
	configFunc configFunc
}

func loadConfiguration(manager *filegen.Manager, filename string) error {
	if filename == "" {
		return nil
	}
	lines, err := fsutil.LoadLines(filename)
	if err != nil {
		return fmt.Errorf("error loading configuration file: %s", err)
	}
	for _, line := range lines {
		fields := strings.Fields(line)
		_ = fields // TODO(rgooch): Implement.
	}
	return nil
}
