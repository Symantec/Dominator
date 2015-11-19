package scanner

import (
	"github.com/Symantec/tricorder/go/tricorder"
	"github.com/Symantec/tricorder/go/tricorder/units"
)

var latencyBucketer *tricorder.Bucketer
var scanTimeDistribution *tricorder.Distribution

func init() {
	latencyBucketer = tricorder.NewGeometricBucketer(1, 10e3)
	scanTimeDistribution = latencyBucketer.NewDistribution()
	tricorder.RegisterMetric("/scan-time", scanTimeDistribution,
		units.Second, "scan time")
}

func (configuration *Configuration) registerMetrics(
	dir *tricorder.DirectorySpec) error {
	netDir, err := dir.RegisterDirectory("network")
	if err != nil {
		return err
	}
	return configuration.NetworkReaderContext.RegisterMetrics(netDir,
		units.Byte, "network speed")
}
