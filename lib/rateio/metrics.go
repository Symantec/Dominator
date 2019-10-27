package rateio

import (
	"github.com/Cloud-Foundations/tricorder/go/tricorder"
	"github.com/Cloud-Foundations/tricorder/go/tricorder/units"
)

var sleepBucketer *tricorder.Bucketer

func init() {
	sleepBucketer = tricorder.NewGeometricBucketer(1e-3, 1e3)
}

func (ctx *ReaderContext) registerMetrics(dir *tricorder.DirectorySpec,
	unit units.Unit, description string) error {
	err := dir.RegisterMetric("speed", &ctx.maxIOPerSecond, unit, description)
	if err != nil {
		return err
	}
	err = dir.RegisterMetric("limit", &ctx.speedPercent, units.None,
		"percent limit")
	if err != nil {
		return err
	}
	ctx.sleepTimeDistribution = sleepBucketer.NewCumulativeDistribution()
	return dir.RegisterMetric("sleep-time", ctx.sleepTimeDistribution,
		units.Millisecond, "sleep time")
}
