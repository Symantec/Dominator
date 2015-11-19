package rateio

import (
	"github.com/Symantec/tricorder/go/tricorder"
	"github.com/Symantec/tricorder/go/tricorder/units"
)

func (ctx *ReaderContext) registerMetrics(dir *tricorder.DirectorySpec,
	unit units.Unit, description string) error {
	err := dir.RegisterMetric("speed", &ctx.maxIOPerSecond, unit, description)
	if err != nil {
		return err
	}
	return dir.RegisterMetric("limit", &ctx.speedPercent, units.None,
		"percent limit")
}
