package fsstorer

import (
	"encoding/gob"
	"net"
	"os"
	"path/filepath"

	"github.com/Symantec/Dominator/lib/fsutil"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
)

func (s *Storer) deleteVm(hypervisor net.IP, ipAddr string) error {
	if dirname, err := s.getVmDirname(hypervisor, ipAddr); err != nil {
		return err
	} else {
		return os.RemoveAll(dirname)
	}
}

func (s *Storer) getNetHypervisorDirectory(hypervisor net.IP) (string, error) {
	hypervisorIP, err := netIpToIp(hypervisor)
	if err != nil {
		return "", err
	}
	return s.getHypervisorDirectory(hypervisorIP), nil
}

func (s *Storer) getVmDirname(hypervisor net.IP,
	ipAddr string) (string, error) {
	if hDirname, err := s.getNetHypervisorDirectory(hypervisor); err != nil {
		return "", err
	} else {
		return filepath.Join(hDirname, "VMs", ipAddr), nil
	}
}

func (s *Storer) listVMs(hypervisor net.IP) ([]string, error) {
	if hDirname, err := s.getNetHypervisorDirectory(hypervisor); err != nil {
		return nil, err
	} else {
		return fsutil.ReadDirnames(filepath.Join(hDirname, "VMs"), true)
	}
}

func (s *Storer) readVm(hypervisor net.IP,
	ipAddr string) (*proto.VmInfo, error) {
	if dirname, err := s.getVmDirname(hypervisor, ipAddr); err != nil {
		return nil, err
	} else {
		filename := filepath.Join(dirname, "info.gob")
		if file, err := os.Open(filename); err != nil {
			return nil, err
		} else {
			defer file.Close()
			decoder := gob.NewDecoder(file)
			var vmInfo proto.VmInfo
			if err := decoder.Decode(&vmInfo); err != nil {
				return nil, err
			}
			return &vmInfo, nil
		}
	}
}

func (s *Storer) writeVm(hypervisor net.IP, ipAddr string,
	vmInfo proto.VmInfo) error {
	if dirname, err := s.getVmDirname(hypervisor, ipAddr); err != nil {
		return err
	} else {
		if err := os.MkdirAll(dirname, dirPerms); err != nil {
			return err
		}
		filename := filepath.Join(dirname, "info.gob")
		writer, err := fsutil.CreateRenamingWriter(filename, filePerms)
		if err != nil {
			return err
		} else {
			defer writer.Close()
			encoder := gob.NewEncoder(writer)
			if err := encoder.Encode(vmInfo); err != nil {
				return err
			}
			return writer.Close()
		}
	}
}
