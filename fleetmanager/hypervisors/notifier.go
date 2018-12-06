package hypervisors

import (
	"bytes"
	"flag"
	"fmt"
	"net/smtp"
	"time"
)

var (
	emailDomain = flag.String("emailDomain", "",
		"Email domain to sent notifications to")
	smtpServer = flag.String("smtpServer", "", "Address of SMTP server")
)

func sendEmail(user string, vms []*vmInfoType) error {
	fromAddress := "DoNotReply@" + *emailDomain
	toAddress := user + "@" + *emailDomain
	buffer := &bytes.Buffer{}
	fmt.Fprintf(buffer, "From: %s\n", fromAddress)
	fmt.Fprintf(buffer, "To: %s\n", toAddress)
	fmt.Fprintln(buffer, "Subject: Please migrate your VMs")
	fmt.Fprintln(buffer)
	fmt.Fprintln(buffer,
		"You own the following VMs which are on unhealthy Hypervisors.")
	fmt.Fprintln(buffer,
		"Please migrate your VMs to healthy Hypervisors ASAP.")
	fmt.Fprintln(buffer, "Below is the list of your VMs which are affected:")
	fmt.Fprintln(buffer)
	for _, vm := range vms {
		fmt.Fprintf(buffer, "IP: %s  name: %s  Hypervisor: %s  status: %s\n",
			vm.Address.IpAddress, vm.Tags["Name"],
			vm.hypervisor.machine.Hostname, vm.hypervisor.getHealthStatus())
	}
	return smtp.SendMail(*smtpServer, nil, fromAddress, []string{toAddress},
		buffer.Bytes())
}

func (h *hypervisorType) addVmsToMap(vmsPerOwner map[string][]*vmInfoType) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	for _, vm := range h.vms {
		for _, owner := range vm.OwnerUsers {
			vmsPerOwner[owner] = append(vmsPerOwner[owner], vm)
		}
	}
}

func (m *Manager) getBadHypervisors() []*hypervisorType {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	badHypervisors := make([]*hypervisorType, 0)
	for _, hypervisor := range m.hypervisors {
		switch hypervisor.probeStatus {
		case probeStatusNotYetProbed:
			continue
		case probeStatusConnected:
			if hypervisor.healthStatus == "" {
				continue
			}
			if hypervisor.healthStatus == "healthy" {
				continue
			}
			badHypervisors = append(badHypervisors, hypervisor)
		default:
			badHypervisors = append(badHypervisors, hypervisor)
		}
	}
	return badHypervisors
}

func (m *Manager) notifierLoop() {
	if *emailDomain == "" || *smtpServer == "" {
		return
	}
	for time.Sleep(time.Minute); ; time.Sleep(time.Hour * 48) {
		m.notify()
	}
}

func (m *Manager) notify() {
	badHypervisors := m.getBadHypervisors()
	if len(badHypervisors) < 1 {
		return
	}
	vmsPerOwner := make(map[string][]*vmInfoType)
	for _, hypervisor := range badHypervisors {
		hypervisor.addVmsToMap(vmsPerOwner)
	}
	for user, vms := range vmsPerOwner {
		if err := sendEmail(user, vms); err != nil {
			m.logger.Printf("error sending email for %s: %s\n", user, err)
		}
	}
}
