package util

import (
	"fmt"
	"github.com/Symantec/Dominator/lib/filegen"
	"github.com/Symantec/Dominator/lib/fsutil"
	"strings"
)

type configFunc func(*filegen.Manager, string, []string) error

type configType struct {
	minParams  int
	maxParams  int
	configFunc configFunc
}

var configs = map[string]configType{
	"DynamicTemplateFile": {1, 1, dynamicTemplateFileGenerator},
	"File":                {1, 1, fileGenerator},
	"StaticTemplateFile":  {1, 1, staticTemplateFileGenerator},
}

func loadConfiguration(manager *filegen.Manager, filename string) error {
	lines, err := fsutil.LoadLines(filename)
	if err != nil {
		return fmt.Errorf("error loading configuration file: %s", err)
	}
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return fmt.Errorf("insufficient fields in line: \"%s\"", line)
		}
		config, ok := configs[fields[0]]
		if !ok {
			return fmt.Errorf("unknown generator type: %s", fields[0])
		}
		numParams := len(fields) - 2
		if numParams < config.minParams {
			return fmt.Errorf("insufficient params in line: \"%s\"", line)
		}
		if config.maxParams >= 0 && numParams > config.maxParams {
			return fmt.Errorf("too many params in line: \"%s\"", line)
		}
		if err := config.configFunc(manager, fields[1],
			fields[2:]); err != nil {
			return err
		}
	}
	return nil
}

func dynamicTemplateFileGenerator(manager *filegen.Manager, pathname string,
	params []string) error {
	return manager.RegisterTemplateFileForPath(pathname, params[0], true)
}

func fileGenerator(manager *filegen.Manager, pathname string,
	params []string) error {
	manager.RegisterFileForPath(pathname, params[0])
	return nil
}

func staticTemplateFileGenerator(manager *filegen.Manager, pathname string,
	params []string) error {
	return manager.RegisterTemplateFileForPath(pathname, params[0], false)
}
