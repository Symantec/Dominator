package resourcepool

import (
	"github.com/Cloud-Foundations/tricorder/go/tricorder"
	"github.com/Cloud-Foundations/tricorder/go/tricorder/units"
)

func (pool *Pool) registerMetrics(metricsSubDirname string) {
	if metricsSubDirname == "" {
		return
	}
	dir, err := tricorder.RegisterDirectory("resourcepool/" + metricsSubDirname)
	if err != nil {
		panic(err)
	}
	err = dir.RegisterMetric("maximum", &pool.max, units.None,
		"maximum capacity")
	if err != nil {
		panic(err)
	}
	err = dir.RegisterMetric("num-in-use", &pool.numUsed, units.None,
		"number in use")
	if err != nil {
		panic(err)
	}
	err = dir.RegisterMetric("num-unused", &pool.numUnused, units.None,
		"number in use")
	if err != nil {
		panic(err)
	}
	err = dir.RegisterMetric("num-releasing", &pool.numReleasing, units.None,
		"number being released")
	if err != nil {
		panic(err)
	}
}
