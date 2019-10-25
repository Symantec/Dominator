package fsstorer

import (
	"bufio"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"

	"github.com/Cloud-Foundations/Dominator/lib/fsutil"
	"github.com/Cloud-Foundations/Dominator/lib/tags"
)

var zeroIP = IP{}

func readIpList(filename string) ([]IP, error) {
	if file, err := os.Open(filename); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	} else {
		defer file.Close()
		reader := bufio.NewReader(file)
		var ipList []IP
		for {
			var ip IP
			if nRead, err := reader.Read(ip[:]); err != nil {
				if err == io.EOF {
					break
				}
				return nil, err
			} else if nRead != len(ip) {
				return nil, errors.New("incomplete read of IP address")
			}
			ipList = append(ipList, ip)
		}
		return ipList, nil
	}
}

func (s *Storer) load() error {
	if err := s.readDirectory(nil, s.topDir); err != nil {
		return err
	}
	for hypervisor, ipAddrs := range s.hypervisorToIPs {
		for _, ipAddr := range ipAddrs {
			s.ipToHypervisor[ipAddr] = hypervisor
		}
	}
	return nil
}

func (s *Storer) readDirectory(partialIP []byte, dirname string) error {
	if len(partialIP) == len(zeroIP) {
		filename := filepath.Join(dirname, "ip-list.raw")
		if ipList, err := readIpList(filename); err != nil {
			return err
		} else {
			var hyperAddr IP
			copy(hyperAddr[:], partialIP)
			s.hypervisorToIPs[hyperAddr] = ipList
		}
		return nil
	}
	names, err := fsutil.ReadDirnames(dirname, true)
	if err != nil {
		return err
	}
	for _, name := range names {
		if val, err := strconv.ParseUint(name, 10, 8); err != nil {
			continue
		} else {
			moreIP := make([]byte, len(partialIP), len(partialIP)+1)
			copy(moreIP, partialIP)
			moreIP = append(moreIP, byte(val))
			subdir := filepath.Join(dirname, name)
			if err := s.readDirectory(moreIP, subdir); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Storer) readMachineSerialNumber(hypervisor net.IP) (string, error) {
	hypervisorIP, err := netIpToIp(hypervisor)
	if err != nil {
		return "", err
	}
	dirname := s.getHypervisorDirectory(hypervisorIP)
	filename := filepath.Join(dirname, "serial-number")
	if file, err := os.Open(filename); err != nil {
		if !os.IsNotExist(err) {
			return "", err
		}
		return "", nil
	} else {
		defer file.Close()
		reader := bufio.NewReader(file)
		if line, err := reader.ReadString('\n'); err != nil {
			return "", err
		} else {
			return line[:len(line)-1], nil
		}
	}
}

func (s *Storer) readMachineTags(hypervisor net.IP) (tags.Tags, error) {
	hypervisorIP, err := netIpToIp(hypervisor)
	if err != nil {
		return nil, err
	}
	dirname := s.getHypervisorDirectory(hypervisorIP)
	filename := filepath.Join(dirname, "tags.raw")
	if file, err := os.Open(filename); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		return nil, nil
	} else {
		defer file.Close()
		reader := bufio.NewReader(file)
		var lastKey string
		tgs := make(tags.Tags)
		for {
			if line, err := reader.ReadString('\n'); err != nil {
				if err != io.EOF {
					return nil, err
				}
				if lastKey != "" {
					return nil, errors.New("missing value for key: " + lastKey)
				}
				break
			} else {
				line = line[:len(line)-1]
				if lastKey == "" {
					lastKey = line
				} else {
					tgs[lastKey] = line
					lastKey = ""
				}
			}
		}
		return tgs, nil
	}
}
