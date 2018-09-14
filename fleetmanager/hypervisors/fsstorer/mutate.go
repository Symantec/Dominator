package fsstorer

import (
	"bufio"
	"errors"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"syscall"

	"github.com/Symantec/Dominator/lib/fsutil"
	"github.com/Symantec/Dominator/lib/tags"
)

const (
	dirPerms = syscall.S_IRWXU | syscall.S_IRGRP | syscall.S_IXGRP |
		syscall.S_IROTH | syscall.S_IXOTH
	filePerms = syscall.S_IRUSR | syscall.S_IWUSR | syscall.S_IRGRP |
		syscall.S_IROTH
)

func (s *Storer) addIPsForHypervisor(hypervisor net.IP,
	netAddrs []net.IP) error {
	hypervisorIP, err := netIpToIp(hypervisor)
	if err != nil {
		return err
	}
	addrs := make([]IP, 0, len(netAddrs))
	for _, addr := range netAddrs {
		if ip, err := netIpToIp(addr); err != nil {
			return err
		} else {
			addrs = append(addrs, ip)
		}
	}
	newAddrs := make([]IP, 0, len(addrs))
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for _, addr := range addrs {
		if hIP, ok := s.ipToHypervisor[addr]; !ok {
			s.ipToHypervisor[addr] = hypervisorIP
			newAddrs = append(newAddrs, addr)
		} else {
			if hIP != hypervisorIP {
				return errors.New("cannot move IP between hypervisors")
			}
		}
	}
	if len(newAddrs) < 1 {
		return nil // No changes.
	}
	err = s.writeIPsForHypervisor(hypervisorIP, addrs, os.O_APPEND)
	if err != nil {
		for _, addr := range newAddrs {
			delete(s.ipToHypervisor, addr)
		}
		return err
	}
	s.hypervisorToIPs[hypervisorIP] = append(s.hypervisorToIPs[hypervisorIP],
		newAddrs...)
	return nil
}

func (s *Storer) getHypervisorDirectory(hypervisor IP) string {
	return filepath.Join(s.topDir,
		strconv.FormatUint(uint64(hypervisor[0]), 10),
		strconv.FormatUint(uint64(hypervisor[1]), 10),
		strconv.FormatUint(uint64(hypervisor[2]), 10),
		strconv.FormatUint(uint64(hypervisor[3]), 10))
}

func (s *Storer) setIPsForHypervisor(hypervisor net.IP,
	netAddrs []net.IP) error {
	hypervisorIP, err := netIpToIp(hypervisor)
	if err != nil {
		return err
	}
	addrs := make([]IP, 0, len(netAddrs))
	for _, addr := range netAddrs {
		if ip, err := netIpToIp(addr); err != nil {
			return err
		} else {
			addrs = append(addrs, ip)
		}
	}
	addrsToForget := make(map[IP]struct{})
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for _, addr := range s.hypervisorToIPs[hypervisorIP] {
		addrsToForget[addr] = struct{}{}
	}
	addedSome := false
	for _, addr := range addrs {
		delete(addrsToForget, addr)
		if hIP, ok := s.ipToHypervisor[addr]; !ok {
			s.ipToHypervisor[addr] = hypervisorIP
			addedSome = true
		} else {
			if hIP != hypervisorIP {
				return errors.New("cannot move IP between hypervisors")
			}
		}
	}
	if !addedSome && len(addrsToForget) < 1 {
		return nil // No changes.
	}
	err = s.writeIPsForHypervisor(hypervisorIP, addrs, os.O_TRUNC)
	if err != nil {
		return err
	}
	for addr := range addrsToForget {
		delete(s.ipToHypervisor, addr)
	}
	s.hypervisorToIPs[hypervisorIP] = addrs
	return nil
}

func (s *Storer) unregisterHypervisor(hypervisor net.IP) error {
	hypervisorIP, err := netIpToIp(hypervisor)
	if err != nil {
		return err
	}
	dirname := s.getHypervisorDirectory(hypervisorIP)
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if err := os.RemoveAll(dirname); err != nil {
		return err
	}
	for _, ip := range s.hypervisorToIPs[hypervisorIP] {
		delete(s.ipToHypervisor, ip)
	}
	delete(s.hypervisorToIPs, hypervisorIP)
	return nil
}

func (s *Storer) writeIPsForHypervisor(hypervisor IP, ipList []IP,
	flags int) error {
	dirname := s.getHypervisorDirectory(hypervisor)
	if dirfile, err := os.Open(dirname); err != nil {
		if err := os.MkdirAll(dirname, dirPerms); err != nil {
			return err
		}
	} else {
		dirfile.Close()
	}
	return writeIpList(filepath.Join(dirname, "ip-list.raw"), ipList, flags)
}

func (s *Storer) writeMachineTags(hypervisor net.IP, tgs tags.Tags) error {
	hypervisorIP, err := netIpToIp(hypervisor)
	if err != nil {
		return err
	}
	dirname := s.getHypervisorDirectory(hypervisorIP)
	filename := filepath.Join(dirname, "tags.raw")
	if len(tgs) < 1 {
		if err := os.Remove(filename); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	keys := make([]string, 0, len(tgs))
	for key := range tgs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	file, err := fsutil.CreateRenamingWriter(filename, filePerms)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	for _, key := range keys {
		if _, err := writer.WriteString(key + "\n"); err != nil {
			return err
		}
		if _, err := writer.WriteString(tgs[key] + "\n"); err != nil {
			return err
		}
	}
	return writer.Flush()
}

func writeIpList(filename string, ipList []IP, flags int) error {
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|flags, filePerms)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	defer writer.Flush()
	for _, ip := range ipList {
		if nWritten, err := writer.Write(ip[:]); err != nil {
			return err
		} else if nWritten != len(ip) {
			return errors.New("short write")
		}
	}
	if err := writer.Flush(); err != nil {
		return err
	}
	return file.Close()
}
