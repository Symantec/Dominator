// +build linux

package main

import (
	"path/filepath"
	"time"

	"github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/slavedriver"
	"github.com/Symantec/Dominator/lib/slavedriver/smallstack"
	"github.com/Symantec/Dominator/proto/hypervisor"
)

type slaveDriverConfiguration struct {
	MaximumIdleSlaves uint
	MinimumIdleSlaves uint
	ImageIdentifier   string
	MemoryInMiB       uint64
	MilliCPUs         uint
}

func createSlaveDriver(logger log.DebugLogger) (
	*slavedriver.SlaveDriver, error) {
	if *slaveDriverConfigurationFile == "" {
		return nil, nil
	}
	var configuration slaveDriverConfiguration
	err := json.ReadFromFile(*slaveDriverConfigurationFile, &configuration)
	if err != nil {
		return nil, err
	}
	slaveTrader, err := smallstack.NewSlaveTrader(hypervisor.CreateVmRequest{
		DhcpTimeout:      time.Minute,
		MinimumFreeBytes: 256 << 20,
		SkipBootloader:   true,
		VmInfo: hypervisor.VmInfo{
			ImageName:   configuration.ImageIdentifier,
			MemoryInMiB: configuration.MemoryInMiB,
			MilliCPUs:   configuration.MilliCPUs,
		},
	}, logger)
	if err != nil {
		return nil, err
	}
	slaveDriver, err := slavedriver.NewSlaveDriver(
		slavedriver.SlaveDriverOptions{
			DatabaseFilename:  filepath.Join(*stateDir, "build-slaves.json"),
			MaximumIdleSlaves: configuration.MaximumIdleSlaves,
			MinimumIdleSlaves: configuration.MinimumIdleSlaves,
			PortNumber:        *portNum,
			Purpose:           "building",
		},
		slaveTrader, logger)
	if err != nil {
		return nil, err
	}
	return slaveDriver, nil
}
