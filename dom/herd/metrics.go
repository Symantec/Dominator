package herd

import (
	"github.com/Symantec/tricorder/go/tricorder"
	"github.com/Symantec/tricorder/go/tricorder/units"
)

var latencyBucketer *tricorder.Bucketer
var shortPollDistribution *tricorder.Distribution
var fullPollDistribution *tricorder.Distribution

func init() {
	latencyBucketer = tricorder.NewExponentialBucketer(20, 1, 1.7)
	shortPollDistribution = tricorder.NewDistribution(latencyBucketer)
	tricorder.RegisterMetric("/poll-short-latency", shortPollDistribution,
		units.Millisecond, "short poll duration")
	fullPollDistribution = tricorder.NewDistribution(latencyBucketer)
	tricorder.RegisterMetric("/poll-full-latency", fullPollDistribution,
		units.Millisecond, "full poll duration")
}
