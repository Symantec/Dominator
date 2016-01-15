package herd

import (
	"github.com/Symantec/tricorder/go/tricorder"
	"github.com/Symantec/tricorder/go/tricorder/units"
)

var latencyBucketer *tricorder.Bucketer
var computeCpuTimeDistribution *tricorder.CumulativeDistribution
var connectDistribution *tricorder.CumulativeDistribution
var fullPollDistribution *tricorder.CumulativeDistribution
var shortPollDistribution *tricorder.CumulativeDistribution
var mdbUpdateTimeDistribution *tricorder.CumulativeDistribution

func init() {
	latencyBucketer = tricorder.NewGeometricBucketer(0.1, 100e3)
	computeCpuTimeDistribution = latencyBucketer.NewCumulativeDistribution()
	tricorder.RegisterMetric("/compute-cputime", computeCpuTimeDistribution,
		units.Millisecond, "compute CPU time")
	connectDistribution = latencyBucketer.NewCumulativeDistribution()
	tricorder.RegisterMetric("/connect-latency", connectDistribution,
		units.Millisecond, "connect duration")
	fullPollDistribution = latencyBucketer.NewCumulativeDistribution()
	tricorder.RegisterMetric("/poll-full-latency", fullPollDistribution,
		units.Millisecond, "full poll duration")
	mdbUpdateTimeDistribution = latencyBucketer.NewCumulativeDistribution()
	tricorder.RegisterMetric("/mdb-update-time", mdbUpdateTimeDistribution,
		units.Millisecond, "time to update Herd MDB data")
	shortPollDistribution = latencyBucketer.NewCumulativeDistribution()
	tricorder.RegisterMetric("/poll-short-latency", shortPollDistribution,
		units.Millisecond, "short poll duration")
}
