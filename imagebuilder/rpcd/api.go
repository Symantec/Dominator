package rpcd

import (
	"io"

	"github.com/Cloud-Foundations/Dominator/imagebuilder/builder"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/lib/srpc/serverutil"
)

type srpcType struct {
	builder *builder.Builder
	logger  log.Logger
	*serverutil.PerUserMethodLimiter
}

type htmlWriter srpcType

func (hw *htmlWriter) WriteHtml(writer io.Writer) {
	hw.writeHtml(writer)
}

func Setup(builder *builder.Builder, logger log.Logger) (*htmlWriter, error) {
	srpcObj := &srpcType{
		builder: builder,
		logger:  logger,
		PerUserMethodLimiter: serverutil.NewPerUserMethodLimiter(
			map[string]uint{
				"BuildImage": 1,
			}),
	}
	srpc.RegisterNameWithOptions("Imaginator", srpcObj,
		srpc.ReceiverOptions{
			PublicMethods: []string{
				"BuildImage",
			}})
	return (*htmlWriter)(srpcObj), nil
}
