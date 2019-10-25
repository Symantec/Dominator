package hypervisors

import (
	"os"

	"github.com/Cloud-Foundations/Dominator/lib/html"
)

func newManager(startOptions StartOptions) (*Manager, error) {
	if err := checkPoolLimits(); err != nil {
		return nil, err
	}
	if startOptions.IpmiPasswordFile != "" {
		file, err := os.Open(startOptions.IpmiPasswordFile)
		if err != nil {
			return nil, err
		}
		file.Close()
	}
	manager := &Manager{
		ipmiUsername:     startOptions.IpmiUsername,
		ipmiPasswordFile: startOptions.IpmiPasswordFile,
		logger:           startOptions.Logger,
		storer:           startOptions.Storer,
		allocatingIPs:    make(map[string]struct{}),
		hypervisors:      make(map[string]*hypervisorType),
		migratingIPs:     make(map[string]struct{}),
		subnets:          make(map[string]*subnetType),
		vms:              make(map[string]*vmInfoType),
	}
	manager.initInvertTable()
	html.HandleFunc("/listHypervisors", manager.listHypervisorsHandler)
	html.HandleFunc("/listLocations", manager.listLocationsHandler)
	html.HandleFunc("/listVMs", manager.listVMsHandler)
	html.HandleFunc("/showHypervisor", manager.showHypervisorHandler)
	go manager.notifierLoop()
	return manager, nil
}
