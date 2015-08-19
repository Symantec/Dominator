package fsrateio

import (
	"github.com/Symantec/Dominator/lib/rateio"
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

func (ctx *ReaderContext) String() string {
	return ctx.format()
}
