package herd

import (
	"time"

	"github.com/Cloud-Foundations/Dominator/lib/cpusharer"
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

func (herd *Herd) setupMetrics(dir *tricorder.DirectorySpec) {
	makeCpuSharerMetrics(dir, "cpu-sharer", herd.cpuSharer)
	latencyBucketer = tricorder.NewGeometricBucketer(0.1, 100e3)
	computeCpuTimeDistribution = makeMetric(dir, latencyBucketer,
		"compute-cputime", "compute CPU time")
	computeTimeDistribution = makeMetric(dir, latencyBucketer,
		"compute-time", "compute time")
	connectDistribution = makeMetric(dir, latencyBucketer,
		"connect-latency", "connect duration")
	mdbUpdateTimeDistribution = makeMetric(dir, latencyBucketer,
		"mdb-update-time", "time to update Herd MDB data")
	fullPollDistribution = makeMetric(dir, latencyBucketer,
		"poll-full-latency", "full poll duration")
	shortPollDistribution = makeMetric(dir, latencyBucketer,
		"poll-short-latency", "short poll duration")
	pollWaitTimeDistribution = makeMetric(dir, latencyBucketer,
		"poll-wait-time", "poll wait time")
}

func makeMetric(dir *tricorder.DirectorySpec, bucketer *tricorder.Bucketer,
	name string, comment string) *tricorder.CumulativeDistribution {
	distribution := latencyBucketer.NewCumulativeDistribution()
	dir.RegisterMetric(name, distribution, units.Millisecond, comment)
	return distribution
}

func makeCpuSharerMetrics(dir *tricorder.DirectorySpec, name string,
	cpuSharer *cpusharer.FifoCpuSharer) {
	dir, err := dir.RegisterDirectory(name)
	if err != nil {
		panic(err)
	}
	group := tricorder.NewGroup()
	group.RegisterUpdateFunc(func() time.Time {
		cpuSharer.GetStatistics()
		return time.Now()
	})
	dir.RegisterMetricInGroup("num-cpu", &cpuSharer.Statistics.NumCpu, group,
		units.None, "number of CPUs")
	dir.RegisterMetricInGroup("num-idle-events",
		&cpuSharer.Statistics.NumIdleEvents, group, units.None,
		"number of CPU idle events")
	dir.RegisterMetricInGroup("num-running",
		&cpuSharer.Statistics.NumCpuRunning, group, units.None,
		"number of running goroutines")
}
