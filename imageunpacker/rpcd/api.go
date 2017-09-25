package rpcd

import (
	"io"
	"sync"

	"github.com/Symantec/Dominator/imageunpacker/unpacker"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
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
