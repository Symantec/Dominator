package herd

import (
	"github.com/Symantec/tricorder/go/tricorder"
	"github.com/Symantec/tricorder/go/tricorder/units"
)

var latencyBucketer *tricorder.Bucketer
var computeCpuTimeDistribution *tricorder.Distribution
var fullPollDistribution *tricorder.Distribution
var shortPollDistribution *tricorder.Distribution
var mdbUpdateTimeDistribution *tricorder.Distribution

func init() {
	latencyBucketer = tricorder.NewGeometricBucketer(0.1, 100e3)
	computeCpuTimeDistribution = latencyBucketer.NewDistribution()
	tricorder.RegisterMetric("/compute-cputime", computeCpuTimeDistribution,
		units.Millisecond, "compute CPU time")
	fullPollDistribution = latencyBucketer.NewDistribution()
	tricorder.RegisterMetric("/poll-full-latency", fullPollDistribution,
		units.Millisecond, "full poll duration")
	mdbUpdateTimeDistribution = latencyBucketer.NewDistribution()
	tricorder.RegisterMetric("/mdb-update-time", mdbUpdateTimeDistribution,
		units.Millisecond, "time to update Herd MDB data")
	shortPollDistribution = latencyBucketer.NewDistribution()
	tricorder.RegisterMetric("/poll-short-latency", shortPollDistribution,
		units.Millisecond, "short poll duration")
}
