package rateio

import (
	"io"
	"time"

	"github.com/Symantec/tricorder/go/tricorder"
	"github.com/Symantec/tricorder/go/tricorder/units"
)

type ReadIOMeasurer interface {
	MeasureReadIO(bytesSinceLastMeasurement uint64) (uint64, error)
	Reset()
}

type ReadMeasurer struct{}

func (measurer *ReadMeasurer) MeasureReadIO(bytesSinceLastMeasurement uint64) (
	uint64, error) {
	return bytesSinceLastMeasurement, nil
}

func (measurer *ReadMeasurer) Reset() {}

type ReaderContext struct {
	maxIOPerSecond        uint64
	speedPercent          uint64
	measurer              ReadIOMeasurer
	bytesSinceLastPause   uint64
	chunklen              uint64
	timeOfLastPause       time.Time
	sleepTimeDistribution *tricorder.CumulativeDistribution
}

func NewReaderContext(maxIOPerSecond uint64, speedPercent uint64,
	measurer ReadIOMeasurer) *ReaderContext {
	return newReaderContext(maxIOPerSecond, speedPercent, measurer)
}

func (ctx *ReaderContext) InitialiseMaximumSpeed(maxSpeed uint64) {
	ctx.initialiseMaximumSpeed(maxSpeed)
}

func (ctx *ReaderContext) MaximumSpeed() uint64 { return ctx.maxIOPerSecond }

func (ctx *ReaderContext) SpeedPercent() uint { return uint(ctx.speedPercent) }

func (ctx *ReaderContext) SetSpeedPercent(percent uint) {
	ctx.setSpeedPercent(percent)
}

func (ctx *ReaderContext) NewReader(rd io.Reader) *Reader {
	return ctx.newReader(rd)
}

func (ctx *ReaderContext) RegisterMetrics(dir *tricorder.DirectorySpec,
	unit units.Unit, description string) error {
	return ctx.registerMetrics(dir, unit, description)
}

func (ctx *ReaderContext) String() string {
	return ctx.format()
}

type Reader struct {
	ctx *ReaderContext
	rd  io.Reader
}

func (rd *Reader) Read(b []byte) (n int, err error) {
	return rd.read(b)
}
