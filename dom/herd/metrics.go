package herd

import (
	"github.com/Symantec/tricorder/go/tricorder"
	"github.com/Symantec/tricorder/go/tricorder/units"
)

var latencyBucketer *tricorder.Bucketer
var computeCpuTimeDistribution *tricorder.CumulativeDistribution
var computeTimeDistribution *tricorder.CumulativeDistribution
var connectDistribution *tricorder.CumulativeDistribution
var mdbUpdateTimeDistribution *tricorder.CumulativeDistribution
var fullPollDistribution *tricorder.CumulativeDistribution
var shortPollDistribution *tricorder.CumulativeDistribution
var pollWaitTimeDistribution *tricorder.CumulativeDistribution

func init() {
	latencyBucketer = tricorder.NewGeometricBucketer(0.1, 100e3)
	computeCpuTimeDistribution = makeMetric(latencyBucketer,
		"/compute-cputime", "compute CPU time")
	computeTimeDistribution = makeMetric(latencyBucketer,
		"/compute-time", "compute time")
	connectDistribution = makeMetric(latencyBucketer,
		"/connect-latency", "connect duration")
	mdbUpdateTimeDistribution = makeMetric(latencyBucketer,
		"/mdb-update-time", "time to update Herd MDB data")
	fullPollDistribution = makeMetric(latencyBucketer,
		"/poll-full-latency", "full poll duration")
	shortPollDistribution = makeMetric(latencyBucketer,
		"/poll-short-latency", "short poll duration")
	pollWaitTimeDistribution = makeMetric(latencyBucketer,
		"/poll-wait-time", "poll wait time")
}

func makeMetric(bucketer *tricorder.Bucketer, name string,
	comment string) *tricorder.CumulativeDistribution {
	distribution := latencyBucketer.NewCumulativeDistribution()
	tricorder.RegisterMetric(name, distribution, units.Millisecond, comment)
	return distribution
}
