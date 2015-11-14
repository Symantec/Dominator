package mdb

import (
	"github.com/Symantec/tricorder/go/tricorder"
	"github.com/Symantec/tricorder/go/tricorder/units"
)

var latencyBucketer *tricorder.Bucketer
var mdbCompareTimeDistribution *tricorder.Distribution
var mdbDecodeTimeDistribution *tricorder.Distribution
var mdbSortTimeDistribution *tricorder.Distribution

func init() {
	latencyBucketer = tricorder.NewGeometricBucketer(0.1, 100e3)
	mdbCompareTimeDistribution = latencyBucketer.NewDistribution()
	tricorder.RegisterMetric("/mdb-compare-time", mdbCompareTimeDistribution,
		units.Millisecond, "time to compare new MDB with old MDB")
	mdbDecodeTimeDistribution = latencyBucketer.NewDistribution()
	tricorder.RegisterMetric("/mdb-decode-time", mdbDecodeTimeDistribution,
		units.Millisecond, "time to decode MDB data")
	mdbSortTimeDistribution = latencyBucketer.NewDistribution()
	tricorder.RegisterMetric("/mdb-sort-time", mdbSortTimeDistribution,
		units.Millisecond, "time to sort MDB data")
}
