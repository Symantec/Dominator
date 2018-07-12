package topology

import (
	"time"

	"github.com/Symantec/Dominator/lib/log"
)

func watch(topologyDir string, checkInterval time.Duration,
	logger log.DebugLogger) (<-chan *Topology, error) {
	topologyChannel := make(chan *Topology, 1)
	go watchLoop(topologyDir, checkInterval, topologyChannel, logger)
	return topologyChannel, nil
}

func watchLoop(topologyDir string, checkInterval time.Duration,
	topologyChannel chan<- *Topology, logger log.DebugLogger) {
	var prevTopology *Topology
	for ; ; time.Sleep(checkInterval) {
		if topology, err := load(topologyDir); err != nil {
			logger.Println(err)
		} else {
			if prevTopology.equal(topology) {
				logger.Debugln(1, "Ignoring unchanged configuration")
			} else {
				topologyChannel <- topology
				prevTopology = topology
			}
		}
	}
}

func (subnet *Subnet) shrink() {
	subnet.Subnet.Shrink()
	for index, ip := range subnet.ReservedIPs {
		if len(ip) == 16 {
			ip = ip.To4()
			if ip != nil {
				subnet.ReservedIPs[index] = ip
			}
		}
	}
}
