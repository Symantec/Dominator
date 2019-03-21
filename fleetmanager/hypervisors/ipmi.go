package hypervisors

import (
	"math/rand"
	"os/exec"
	"strings"
	"time"
)

const powerOff = "Power is off"

func (m *Manager) probeUnreachable(h *hypervisorType) probeStatus {
	if m.ipmiPasswordFile == "" || m.ipmiUsername == "" {
		return probeStatusUnreachable
	}
	var hostname string
	if len(h.machine.IPMI.HostIpAddress) > 0 {
		hostname = h.machine.IPMI.HostIpAddress.String()
	} else if h.machine.IPMI.Hostname != "" {
		hostname = h.machine.IPMI.Hostname
	} else {
		return probeStatusUnreachable
	}
	h.mutex.RLock()
	previousProbeStatus := h.probeStatus
	h.mutex.RUnlock()
	mimimumProbeInterval := time.Second * time.Duration(30+rand.Intn(30))
	if previousProbeStatus == probeStatusOff &&
		time.Until(h.lastIpmiProbe.Add(mimimumProbeInterval)) > 0 {
		return probeStatusOff
	}
	cmd := exec.Command("ipmitool", "-f", m.ipmiPasswordFile, "-H", hostname,
		"-I", "lanplus", "-U", m.ipmiUsername, "chassis", "power", "status")
	h.lastIpmiProbe = time.Now()
	if output, err := cmd.Output(); err != nil {
		if previousProbeStatus == probeStatusOff {
			return probeStatusOff
		} else {
			return probeStatusUnreachable
		}
	} else if strings.Contains(string(output), powerOff) {
		return probeStatusOff
	}
	return probeStatusUnreachable
}
