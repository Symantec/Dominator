package manager

import (
	"bytes"
	"crypto/rand"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/Symantec/Dominator/lib/fsutil"
	"github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/lib/log/prefixlogger"
	"github.com/Symantec/Dominator/lib/meminfo"
	"github.com/Symantec/Dominator/lib/objectserver/cachingreader"
	"github.com/Symantec/Dominator/lib/rpcclientpool"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
	"github.com/Symantec/tricorder/go/tricorder/messages"
	trimsg "github.com/Symantec/tricorder/go/tricorder/messages"
)

const (
	dirPerms = syscall.S_IRWXU | syscall.S_IRGRP | syscall.S_IXGRP |
		syscall.S_IROTH | syscall.S_IXOTH
	privateFilePerms  = syscall.S_IRUSR | syscall.S_IWUSR
	publicFilePerms   = privateFilePerms | syscall.S_IRGRP | syscall.S_IROTH
	productSerialFile = "/sys/class/dmi/id/product_serial"
)

func newManager(startOptions StartOptions) (*Manager, error) {
	memInfo, err := meminfo.GetMemInfo()
	if err != nil {
		return nil, err
	}
	rootCookie := make([]byte, 32)
	if _, err := rand.Read(rootCookie); err != nil {
		return nil, err
	}
	err = fsutil.CopyToFile(filepath.Join(startOptions.StateDir,
		"root-cookie"), privateFilePerms, bytes.NewReader(rootCookie), 0)
	if err != nil {
		return nil, err
	}
	manager := &Manager{
		StartOptions:      startOptions,
		rootCookie:        rootCookie,
		memTotalInMiB:     memInfo.Total >> 20,
		notifiers:         make(map[<-chan proto.Update]chan<- proto.Update),
		numCPU:            runtime.NumCPU(),
		serialNumber:      readProductSerial(),
		vms:               make(map[string]*vmInfoType),
		volumeDirectories: startOptions.VolumeDirectories,
	}
	if err := manager.loadSubnets(); err != nil {
		return nil, err
	}
	if err := manager.loadAddressPool(); err != nil {
		return nil, err
	}
	dirname := filepath.Join(manager.StateDir, "VMs")
	dir, err := os.Open(dirname)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.Mkdir(dirname, dirPerms); err != nil {
				return nil, errors.New(
					"error making: " + dirname + ": " + err.Error())
			}
			dir, err = os.Open(dirname)
		}
	}
	if err != nil {
		return nil, err
	}
	defer dir.Close()
	names, err := dir.Readdirnames(-1)
	if err != nil {
		return nil, errors.New(
			"error reading directory: " + dirname + ": " + err.Error())
	}
	for _, ipAddr := range names {
		vmDirname := filepath.Join(dirname, ipAddr)
		filename := filepath.Join(vmDirname, "info.json")
		var vmInfo vmInfoType
		if err := json.ReadFromFile(filename, &vmInfo); err != nil {
			return nil, err
		}
		vmInfo.Address.Shrink()
		vmInfo.manager = manager
		vmInfo.dirname = vmDirname
		vmInfo.ipAddress = ipAddr
		vmInfo.ownerUsers = make(map[string]struct{}, len(vmInfo.OwnerUsers))
		for _, username := range vmInfo.OwnerUsers {
			vmInfo.ownerUsers[username] = struct{}{}
		}
		vmInfo.logger = prefixlogger.New(ipAddr+": ", manager.Logger)
		vmInfo.metadataChannels = make(map[chan<- string]struct{})
		manager.vms[ipAddr] = &vmInfo
		if _, err := vmInfo.startManaging(0, false); err != nil {
			manager.Logger.Println(err)
			if ipAddr == "0.0.0.0" {
				delete(manager.vms, ipAddr)
				vmInfo.destroy()
			}
		}
	}
	// Check address pool for used addresses with no VM.
	freeIPs := make(map[string]struct{}, len(manager.addressPool.Free))
	for _, addr := range manager.addressPool.Free {
		freeIPs[addr.IpAddress.String()] = struct{}{}
	}
	for _, addr := range manager.addressPool.Registered {
		ipAddr := addr.IpAddress.String()
		if _, ok := freeIPs[ipAddr]; ok {
			continue
		}
		if _, ok := manager.vms[ipAddr]; !ok {
			manager.Logger.Printf("%s shown as used but no corresponding VM\n",
				ipAddr)
		}
	}
	if len(manager.volumeDirectories) < 1 {
		manager.volumeDirectories, err = getVolumeDirectories()
		if err != nil {
			return nil, err
		}
	}
	if len(manager.volumeDirectories) < 1 {
		return nil, errors.New("no volume directories available")
	}
	for _, volumeDirectory := range manager.volumeDirectories {
		if err := os.MkdirAll(volumeDirectory, dirPerms); err != nil {
			return nil, err
		}
	}
	if startOptions.ObjectCacheBytes >= 1<<20 {
		dirname := filepath.Join(filepath.Dir(manager.volumeDirectories[0]),
			"objectcache")
		if err := os.MkdirAll(dirname, dirPerms); err != nil {
			return nil, err
		}
		objSrv, err := cachingreader.NewObjectServer(dirname,
			startOptions.ObjectCacheBytes, startOptions.ImageServerAddress,
			startOptions.Logger)
		if err != nil {
			return nil, err
		}
		manager.objectCache = objSrv
	}
	go manager.loopCheckHealthStatus()
	return manager, nil
}

func readProductSerial() string {
	if file, err := os.Open(productSerialFile); err != nil {
		return ""
	} else {
		defer file.Close()
		buffer := make([]byte, 256)
		if nRead, err := file.Read(buffer); err != nil {
			return ""
		} else if nRead < 1 {
			return ""
		} else {
			serial := strings.TrimSpace(string(buffer[:nRead]))
			if serial == "System Serial Number" {
				serial = ""
			}
			return serial
		}
	}
}

func (m *Manager) loopCheckHealthStatus() {
	cr := rpcclientpool.New("tcp", ":6910", true, "")
	for ; ; time.Sleep(time.Second * 10) {
		healthStatus := m.checkHealthStatus(cr)
		m.mutex.Lock()
		if m.healthStatus != healthStatus {
			m.healthStatus = healthStatus
			m.sendUpdateWithLock(proto.Update{})
		}
		m.mutex.Unlock()
	}
}

func (m *Manager) checkHealthStatus(cr *rpcclientpool.ClientResource) string {
	client, err := cr.Get(nil)
	if err != nil {
		m.Logger.Printf("error getting health-agent client: %s", err)
		return "bad health-agent"
	}
	defer client.Put()
	var metric messages.Metric
	err = client.Call("MetricsServer.GetMetric", "/sys/storage/health", &metric)
	if err != nil {
		if strings.Contains(err.Error(), trimsg.ErrMetricNotFound.Error()) {
			return ""
		}
		m.Logger.Printf("error getting health-agent metrics: %s", err)
		client.Close()
		return "failed getting health metrics"
	}
	if healthStatus, ok := metric.Value.(string); !ok {
		m.Logger.Println("list metric is not string")
		return "bad health metric type"
	} else if healthStatus == "good" {
		return "healthy"
	} else {
		return healthStatus
	}
}
