package hypervisors

import (
	"github.com/Symantec/Dominator/lib/html"
	"github.com/Symantec/Dominator/lib/log"
)

func newManager(storer Storer, logger log.DebugLogger) (*Manager, error) {
	if err := checkPoolLimits(); err != nil {
		return nil, err
	}
	manager := &Manager{
		storer:       storer,
		logger:       logger,
		hypervisors:  make(map[string]*hypervisorType),
		migratingIPs: make(map[string]struct{}),
		subnets:      make(map[string]*subnetType),
		vms:          make(map[string]*vmInfoType),
	}
	manager.initInvertTable()
	html.HandleFunc("/listHypervisors", manager.listHypervisorsHandler)
	html.HandleFunc("/listLocations", manager.listLocationsHandler)
	html.HandleFunc("/listVMs", manager.listVMsHandler)
	html.HandleFunc("/showHypervisor", manager.showHypervisorHandler)
	go manager.notifierLoop()
	return manager, nil
}
