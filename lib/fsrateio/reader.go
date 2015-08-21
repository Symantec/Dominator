package fsrateio

import (
	"fmt"
	"github.com/Symantec/Dominator/lib/format"
	"github.com/Symantec/Dominator/lib/rateio"
	"syscall"
)

type ReadMeasurer struct {
	blocksAtLastMeasurement uint64
}

func newReadMeasurer() *ReadMeasurer {
	var measurer ReadMeasurer
	return &measurer
}

func (measurer *ReadMeasurer) MeasureReadIO(bytesSinceLastMeasurement uint64) (
	uint64, error) {
	var rusage syscall.Rusage
	err := syscall.Getrusage(syscall.RUSAGE_SELF, &rusage)
	if err != nil {
		return 0, err
	}
	blocks := uint64(rusage.Inblock)
	blocksSinceLastMeasurement := blocks - measurer.blocksAtLastMeasurement
	measurer.blocksAtLastMeasurement = blocks
	return blocksSinceLastMeasurement, nil
}

func (measurer *ReadMeasurer) Reset() {
	var rusage syscall.Rusage
	syscall.Getrusage(syscall.RUSAGE_SELF, &rusage)
	measurer.blocksAtLastMeasurement = uint64(rusage.Inblock)
}

func newReaderContext(maxBytesPerSecond uint64, maxBlocksPerSecond uint64,
	speedPercent uint64) *ReaderContext {
	var ctx ReaderContext
	ctx.maxBytesPerSecond = maxBytesPerSecond
	ctx.maxBlocksPerSecond = maxBlocksPerSecond
	if maxBlocksPerSecond > 0 {
		ctx.ctx = rateio.NewReaderContext(maxBlocksPerSecond, speedPercent,
			newReadMeasurer())
	} else {
		ctx.ctx = rateio.NewReaderContext(maxBytesPerSecond, speedPercent,
			&rateio.ReadMeasurer{})
	}
	return &ctx
}

func (ctx *ReaderContext) format() string {
	var blocksString string
	if ctx.maxBlocksPerSecond > 0 {
		blocksString = fmt.Sprintf("(%d blocks/s)", ctx.maxBlocksPerSecond)
	} else {
		blocksString = ""
	}
	speedPercent := uint64(ctx.GetContext().SpeedPercent())
	return fmt.Sprintf("max speed=%s/s%s limit=%d%% %s/s(%d blocks/s)",
		format.FormatBytes(ctx.maxBytesPerSecond), blocksString,
		speedPercent,
		format.FormatBytes(ctx.maxBytesPerSecond*speedPercent/100),
		ctx.maxBlocksPerSecond*speedPercent/100)
}
