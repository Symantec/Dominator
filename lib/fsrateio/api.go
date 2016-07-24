package fsrateio

import (
	"github.com/Symantec/Dominator/lib/rateio"
	"github.com/Symantec/tricorder/go/tricorder"
	"github.com/Symantec/tricorder/go/tricorder/units"
	"io"
)

type ReaderContext struct {
	maxBytesPerSecond  uint64
	maxBlocksPerSecond uint64
	ctx                *rateio.ReaderContext
}

func NewReaderContext(maxBytesPerSecond uint64,
	maxBlocksPerSecond uint64, speedPercent uint64) *ReaderContext {
	return newReaderContext(maxBytesPerSecond, maxBlocksPerSecond, speedPercent)
}

func (ctx *ReaderContext) GetContext() *rateio.ReaderContext { return ctx.ctx }

func (ctx *ReaderContext) NewReader(rd io.Reader) *rateio.Reader {
	return ctx.ctx.NewReader(rd)
}

func (ctx *ReaderContext) RegisterMetrics(dir *tricorder.DirectorySpec) error {
	if ctx.maxBlocksPerSecond > 0 {
		return ctx.ctx.RegisterMetrics(dir, units.None,
			"file-system speed in blocks per second")
	}
	return ctx.ctx.RegisterMetrics(dir, units.BytePerSecond,
		"file-system speed")
}

func (ctx *ReaderContext) String() string {
	return ctx.format()
}
