package rpcd

import (
	"io"

	"github.com/Symantec/Dominator/fleetmanager/hypervisors"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
)

type srpcType struct {
	hypervisorsManager *hypervisors.Manager
	logger             log.DebugLogger
}

type htmlWriter srpcType

func (hw *htmlWriter) WriteHtml(writer io.Writer) {
	hw.writeHtml(writer)
}

func Setup(hypervisorsManager *hypervisors.Manager, logger log.DebugLogger) (
	*htmlWriter, error) {
	srpcObj := &srpcType{
		hypervisorsManager: hypervisorsManager,
		logger:             logger,
	}
	srpc.RegisterNameWithOptions("FleetManager", srpcObj,
		srpc.ReceiverOptions{
			PublicMethods: []string{
				"GetHypervisorForVM",
				"ListHypervisorLocations",
				"ListHypervisorsInLocation",
				"ListVMsInLocation",
			}})
	return (*htmlWriter)(srpcObj), nil
}
