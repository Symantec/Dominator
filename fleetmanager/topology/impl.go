package topology

import (
	"bytes"
	"errors"
	"net"
	"time"

	"github.com/Symantec/Dominator/lib/log"
)

func parseMAC(text []byte) (HardwareAddr, error) {
	text = bytes.ToLower(text)
	buf := make([]byte, 20)
	writePosition := 0
	for _, char := range text {
		var value byte
		if char >= '0' && char <= '9' {
			value = char - '0'
		} else if char >= 'a' && char <= 'f' {
			value = 10 + char - 'a'
		} else if char == ':' {
			writePosition++
			if writePosition >= len(buf) {
				return nil, errors.New("invalid MAC")
			}
			continue
		} else {
			return nil, errors.New("invalid MAC")
		}
		if buf[writePosition]&0xf0 != 0 {
			return nil, errors.New("invalid MAC")
		}
		buf[writePosition] = buf[writePosition]<<4 + value
	}
	addr := make([]byte, writePosition+1) // Make a copy just long enough.
	copy(addr, buf)
	return addr, nil
}

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

func (addr HardwareAddr) MarshalText() (text []byte, err error) {
	return []byte(addr.String()), nil
}

func (addr HardwareAddr) String() string {
	return net.HardwareAddr(addr).String()
}

func (addr *HardwareAddr) UnmarshalText(text []byte) error {
	text = bytes.ToLower(text)
	if newAddr, err := parseMAC(text); err == nil {
		*addr = newAddr
		return nil
	}
	if hw, err := net.ParseMAC(string(text)); err != nil {
		return err
	} else {
		*addr = HardwareAddr(hw)
		return nil
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
