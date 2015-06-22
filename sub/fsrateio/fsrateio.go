package fsrateio

import (
	"fmt"
	"github.com/Symantec/Dominator/lib/fsbench"
	"io"
	"syscall"
	"time"
)

const (
	CHUNKLEN              = 1024 * 1024
	DEFAULT_SPEED_PERCENT = 2
)

type FsRateContext struct {
	maxBytesPerSecond    uint64
	maxBlocksPerSecond   uint64
	speedPercent         uint64
	bytesSinceLastPause  uint64
	blocksSinceLastPause uint64
	timeOfLastPause      time.Time
}

func (ctx *FsRateContext) SpeedPercent() uint {
	return uint(ctx.speedPercent)
}

func (ctx *FsRateContext) SetSpeedPercent(percent uint) {
	if percent > 100 {
		percent = 100
	}
	ctx.speedPercent = uint64(percent)
}

func formatBytesPerSecond(bps uint64) string {
	if bps>>30 > 100 {
		return fmt.Sprintf("%d GiB/s", bps>>30)
	} else if bps>>20 > 100 {
		return fmt.Sprintf("%d MiB/s", bps>>20)
	} else if bps>>10 > 100 {
		return fmt.Sprintf("%d KiB/s", bps>>10)
	} else {
		return fmt.Sprintf("%d B/s", bps)
	}
}

func (ctx *FsRateContext) String() string {
	var blocksString string
	if ctx.maxBlocksPerSecond > 0 {
		blocksString = fmt.Sprintf("(%d blocks/s)", ctx.maxBlocksPerSecond)
	} else {
		blocksString = ""
	}
	return fmt.Sprintf("max speed=%s%s limit=%d%% %s(%d blocks/s)",
		formatBytesPerSecond(ctx.maxBytesPerSecond), blocksString,
		ctx.speedPercent,
		formatBytesPerSecond(ctx.maxBytesPerSecond*ctx.speedPercent/100),
		ctx.maxBlocksPerSecond*ctx.speedPercent/100)
}

func NewContext(benchFile string) (*FsRateContext, error) {
	var ctx FsRateContext
	ctx.SetSpeedPercent(DEFAULT_SPEED_PERCENT)
	var err error
	ctx.maxBytesPerSecond, ctx.maxBlocksPerSecond, err =
		fsbench.GetReadSpeed(benchFile)
	if err != nil {
		return nil, err
	}
	var rusage syscall.Rusage
	syscall.Getrusage(syscall.RUSAGE_SELF, &rusage)
	ctx.blocksSinceLastPause = uint64(rusage.Inblock)
	ctx.timeOfLastPause = time.Now()
	return &ctx, nil
}

type Reader struct {
	ctx *FsRateContext
	rd  io.Reader
}

func NewReader(rd io.Reader, ctx *FsRateContext) *Reader {
	r := new(Reader)
	r.ctx = ctx
	r.rd = rd
	return r
}

func (rd *Reader) Read(b []byte) (n int, err error) {
	if rd.ctx.SpeedPercent() >= 100 {
		// Operate at maximum speed: get out of the way.
		return rd.rd.Read(b)
	}
	if rd.ctx.bytesSinceLastPause >= CHUNKLEN {
		// Need to slow down.
		var desiredPerSecond, readSinceLastPause uint64
		var rusage syscall.Rusage
		if rd.ctx.maxBlocksPerSecond > 0 {
			desiredPerSecond = rd.ctx.maxBlocksPerSecond
			syscall.Getrusage(syscall.RUSAGE_SELF, &rusage)
			readSinceLastPause = uint64(rusage.Inblock) -
				rd.ctx.blocksSinceLastPause
		} else {
			desiredPerSecond = rd.ctx.maxBytesPerSecond
			readSinceLastPause = rd.ctx.bytesSinceLastPause
		}
		desiredPerSecond = desiredPerSecond * rd.ctx.speedPercent / 100
		desiredDuration := time.Duration(uint64(time.Second) *
			uint64(readSinceLastPause) / desiredPerSecond)
		targetTime := rd.ctx.timeOfLastPause.Add(desiredDuration)
		time.Sleep(targetTime.Sub(time.Now()))
		rd.ctx.bytesSinceLastPause = 0
		if rd.ctx.maxBlocksPerSecond > 0 {
			rd.ctx.blocksSinceLastPause += readSinceLastPause
		}
		rd.ctx.timeOfLastPause = time.Now()
	}
	n, err = rd.rd.Read(b)
	if n < 1 || err != nil {
		return
	}
	rd.ctx.bytesSinceLastPause += uint64(n)
	return
}
