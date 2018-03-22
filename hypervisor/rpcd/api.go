package rpcd

import (
	"io"

	"github.com/Symantec/Dominator/hypervisor/manager"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
)

type srpcType struct {
	manager *manager.Manager
	logger  log.DebugLogger
}

type htmlWriter srpcType

func (hw *htmlWriter) WriteHtml(writer io.Writer) {
	hw.writeHtml(writer)
}

func Setup(manager *manager.Manager, logger log.DebugLogger) (
	*htmlWriter, error) {
	srpcObj := &srpcType{
		manager: manager,
		logger:  logger,
	}
	srpc.RegisterNameWithOptions("Hypervisor", srpcObj, srpc.ReceiverOptions{
		PublicMethods: []string{
			"AcknowledgeVm",
			"ChangeVmTags",
			"CreateVm",
			"DestroyVm",
			"ReplaceVmImage",
			"RestoreVmImage",
			"StartVm",
			"StopVm",
		}})
	return (*htmlWriter)(srpcObj), nil
}