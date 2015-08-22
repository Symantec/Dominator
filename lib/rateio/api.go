package rateio

import (
	"io"
	"time"
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
	maxIOPerSecond      uint64
	speedPercent        uint64
	measurer            ReadIOMeasurer
	bytesSinceLastPause uint64
	timeOfLastPause     time.Time
}

func NewReaderContext(maxIOPerSecond uint64, speedPercent uint64,
	measurer ReadIOMeasurer) *ReaderContext {
	return newReaderContext(maxIOPerSecond, speedPercent, measurer)
}

func (ctx *ReaderContext) SpeedPercent() uint { return uint(ctx.speedPercent) }

func (ctx *ReaderContext) SetSpeedPercent(percent uint) {
	ctx.setSpeedPercent(percent)
}

func (ctx *ReaderContext) NewReader(rd io.Reader) *Reader {
	return ctx.newReader(rd)
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
