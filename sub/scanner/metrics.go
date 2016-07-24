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
	return configuration.NetworkReaderContext.RegisterMetrics(netDir,
		units.BytePerSecond, "network speed")
}
