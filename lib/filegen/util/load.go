package util

import (
	"fmt"
	"github.com/Symantec/Dominator/lib/filegen"
	"github.com/Symantec/Dominator/lib/fsutil"
	"strings"
)

type configFunc func(*filegen.Manager, []string) error

type configType struct {
	minParams  int
	maxParams  int
	configFunc configFunc
}

var configs = map[string]configType{}

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
		if len(fields) < 2 {
			continue
		}
		config, ok := configs[fields[0]]
		if !ok {
			continue
		}
		numParams := len(fields) - 1
		if numParams < config.minParams {
			return fmt.Errorf("insufficient params in line: %s", line)
		}
		if config.maxParams >= 0 && numParams > config.maxParams {
			return fmt.Errorf("too many params in line: %s", line)
		}
		// TODO(rgooch): Implement.
	}
	return nil
}
