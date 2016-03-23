package rateio

import (
	"fmt"
	"github.com/Symantec/Dominator/lib/format"
	"io"
	"time"
)

const (
	DEFAULT_SPEED_PERCENT = 2
)

func newReaderContext(maxIOPerSecond uint64, speedPercent uint64,
	measurer ReadIOMeasurer) *ReaderContext {
	var ctx ReaderContext
	ctx.maxIOPerSecond = maxIOPerSecond
	if speedPercent < 1 {
		speedPercent = DEFAULT_SPEED_PERCENT
	}
	ctx.speedPercent = speedPercent
	ctx.chunklen = ctx.maxIOPerSecond * ctx.speedPercent / 10000
	ctx.measurer = measurer
	ctx.timeOfLastPause = time.Now()
	measurer.Reset()
	return &ctx
}

func (ctx *ReaderContext) initialiseMaximumSpeed(maxSpeed uint64) {
	if ctx.maxIOPerSecond > 0 {
		fmt.Println("Maximum speed already set")
	}
	ctx.maxIOPerSecond = maxSpeed
	ctx.chunklen = ctx.maxIOPerSecond * ctx.speedPercent / 10000
}

func (ctx *ReaderContext) setSpeedPercent(percent uint) {
	if percent > 100 {
		percent = 100
	}
	ctx.speedPercent = uint64(percent)
	ctx.chunklen = ctx.maxIOPerSecond * ctx.speedPercent / 10000
	ctx.timeOfLastPause = time.Now()
	ctx.measurer.Reset()
}

func (ctx *ReaderContext) newReader(rd io.Reader) *Reader {
	var reader Reader
	reader.ctx = ctx
	reader.rd = rd
	return &reader
}

func (ctx *ReaderContext) format() string {
	return fmt.Sprintf("max speed=%s/s limit=%d%% %s/s",
		format.FormatBytes(ctx.maxIOPerSecond),
		ctx.speedPercent,
		format.FormatBytes(ctx.maxIOPerSecond*ctx.speedPercent/100))
}

func (rd *Reader) read(b []byte) (n int, err error) {
	if rd.ctx.maxIOPerSecond < 1 {
		// Unspecified capacity: go at maximum speed.
		return rd.rd.Read(b)
	}
	if rd.ctx.speedPercent >= 100 {
		// Operate at maximum speed: get out of the way.
		return rd.rd.Read(b)
	}
	if rd.ctx.bytesSinceLastPause >= rd.ctx.chunklen {
		// Need to slow down.
		desiredPerSecond := rd.ctx.maxIOPerSecond * rd.ctx.speedPercent / 100
		readSinceLastPause, err := rd.ctx.measurer.MeasureReadIO(
			rd.ctx.bytesSinceLastPause)
		if err != nil {
			return 0, err
		}
		desiredDuration := time.Duration(uint64(time.Second) *
			readSinceLastPause / desiredPerSecond)
		targetTime := rd.ctx.timeOfLastPause.Add(desiredDuration)
		time.Sleep(targetTime.Sub(time.Now()))
		rd.ctx.bytesSinceLastPause = 0
		rd.ctx.timeOfLastPause = time.Now()
	}
	n, err = rd.rd.Read(b)
	if n < 1 || err != nil {
		return
	}
	rd.ctx.bytesSinceLastPause += uint64(n)
	return
}
