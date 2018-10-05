package hypervisors

import (
	"net/http"

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
	http.HandleFunc("/listHypervisors", manager.listHypervisorsHandler)
	http.HandleFunc("/listLocations", manager.listLocationsHandler)
	http.HandleFunc("/listVMs", manager.listVMsHandler)
	http.HandleFunc("/showHypervisor", manager.showHypervisorHandler)
	go manager.notifierLoop()
	return manager, nil
}
