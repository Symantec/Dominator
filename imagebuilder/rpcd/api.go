package rpcd

import (
	"github.com/Symantec/Dominator/imagebuilder/builder"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
	"io"
)

type srpcType struct {
	builder *builder.Builder
	logger  log.Logger
}

type htmlWriter srpcType

func (hw *htmlWriter) WriteHtml(writer io.Writer) {
	hw.writeHtml(writer)
}

func Setup(builder *builder.Builder, logger log.Logger) (*htmlWriter, error) {
	srpcObj := &srpcType{
		builder: builder,
		logger:  logger,
	}
	srpc.RegisterName("Imaginator", srpcObj)
	return (*htmlWriter)(srpcObj), nil
}
