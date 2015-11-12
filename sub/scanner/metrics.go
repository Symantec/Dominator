package scanner

import (
	"github.com/Symantec/tricorder/go/tricorder"
	"github.com/Symantec/tricorder/go/tricorder/units"
)

var latencyBucketer *tricorder.Bucketer
var scanTimeDistribution *tricorder.Distribution

func init() {
	latencyBucketer = tricorder.NewExponentialBucketer(20, 1, 1.7)
	scanTimeDistribution = tricorder.NewDistribution(latencyBucketer)
	tricorder.RegisterMetric("/scan-time", scanTimeDistribution,
		units.Second, "scan time")
}
