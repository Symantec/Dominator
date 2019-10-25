package rpcd

import (
	"io"
	"sync"

	"github.com/Cloud-Foundations/Dominator/imageunpacker/unpacker"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
)

type srpcType struct {
	unpacker      *unpacker.Unpacker
	logger        log.Logger
	addDeviceLock sync.Mutex
}

type htmlWriter srpcType

func (hw *htmlWriter) WriteHtml(writer io.Writer) {
	hw.writeHtml(writer)
}

func Setup(unpackerObj *unpacker.Unpacker, logger log.Logger) *htmlWriter {
	srpcObj := srpcType{
		unpacker: unpackerObj,
		logger:   logger}
	srpc.RegisterName("ImageUnpacker", &srpcObj)
	return (*htmlWriter)(&srpcObj)
}
