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
