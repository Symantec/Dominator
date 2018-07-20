package scanner

import (
	"github.com/Symantec/tricorder/go/tricorder"
	"github.com/Symantec/tricorder/go/tricorder/units"
)

var latencyBucketer *tricorder.Bucketer
var scanTimeDistribution *tricorder.CumulativeDistribution

func init() {
	latencyBucketer = tricorder.NewGeometricBucketer(1, 10e3)
	scanTimeDistribution = latencyBucketer.NewCumulativeDistribution()
	tricorder.RegisterMetric("/scan-time", scanTimeDistribution,
		units.Second, "scan time")
}

func (configuration *Configuration) registerMetrics(
	dir *tricorder.DirectorySpec) error {
	scannerDir, err := dir.RegisterDirectory("scanner")
	if err != nil {
		return err
	}
	err = configuration.FsScanContext.RegisterMetrics(scannerDir)
	if err != nil {
		return err
	}
	netDir, err := dir.RegisterDirectory("network")
	if err != nil {
		return err
	}
	err = configuration.NetworkReaderContext.RegisterMetrics(netDir,
		units.BytePerSecond, "network speed")
	if err != nil {
		return err
	}
	if configuration.ScanFilter != nil {
		list := tricorder.NewList(configuration.ScanFilter.FilterLines, false)
		err := scannerDir.RegisterMetric("scan-filter", list, units.None,
			"scan filter lines")
		if err != nil {
			return err
		}
	}
	return nil
}
