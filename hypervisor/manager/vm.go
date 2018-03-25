package manager

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path"
	"syscall"
	"time"

	imclient "github.com/Symantec/Dominator/imageserver/client"
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/filesystem/util"
	"github.com/Symantec/Dominator/lib/fsutil"
	"github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/lib/log/prefixlogger"
	"github.com/Symantec/Dominator/lib/mbr"
	objclient "github.com/Symantec/Dominator/lib/objectserver/client"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/lib/tags"
	"github.com/Symantec/Dominator/lib/verstr"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
)

const (
	privateFilePerms = syscall.S_IRUSR | syscall.S_IWUSR
	publicFilePerms  = privateFilePerms | syscall.S_IRGRP | syscall.S_IROTH
)

var (
	errorNoAccessToResource = errors.New("no access to resource")
)

type sendErrorFunc func(conn *srpc.Conn, encoder srpc.Encoder, err error) error
type sendUpdateFunc func(conn *srpc.Conn, encoder srpc.Encoder,
	message string) error

func computeSize(minimumFreeBytes, roundupPower, size uint64) uint64 {
	minBytes := size + size>>3 // 12% extra for good luck.
	minBytes += minimumFreeBytes
	if roundupPower < 24 {
		roundupPower = 24 // 16 MiB.
	}
	imageUnits := minBytes >> roundupPower
	if imageUnits<<roundupPower < minBytes {
		imageUnits++
	}
	return imageUnits << roundupPower
}

func copyData(filename string, reader io.Reader, length uint64,
	perm os.FileMode) error {
	if length < 1 {
		return nil
	}
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, perm)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	defer writer.Flush()
	_, err = io.CopyN(writer, reader, int64(length))
	return err
}

func getImage(client *srpc.Client, imageName string,
	imageTimeout time.Duration) (*filesystem.FileSystem, error) {
	img, err := imclient.GetImageWithTimeout(client, imageName,
		imageTimeout)
	if err != nil {
		return nil, err
	}
	if img == nil {
		return nil, errors.New("timeout getting image")
	}
	img.FileSystem.RebuildInodePointers()
	return img.FileSystem, nil
}

func maybeDrainAll(conn *srpc.Conn, request proto.CreateVmRequest) error {
	if err := maybeDrainImage(conn, request.ImageDataSize); err != nil {
		return err
	}
	if err := maybeDrainUserData(conn, request); err != nil {
		return err
	}
	return nil
}

func maybeDrainImage(imageReader io.Reader, imageDataSize uint64) error {
	if imageDataSize > 0 { // Drain data.
		_, err := io.CopyN(ioutil.Discard, imageReader, int64(imageDataSize))
		return err
	}
	return nil
}

func maybeDrainUserData(conn *srpc.Conn, request proto.CreateVmRequest) error {
	if request.UserDataSize > 0 { // Drain data.
		_, err := io.CopyN(ioutil.Discard, conn, int64(request.UserDataSize))
		return err
	}
	return nil
}

func setVolumeSize(filename string, size uint64) error {
	if err := os.Truncate(filename, int64(size)); err != nil {
		return err
	}
	return fsutil.Fallocate(filename, size)
}

func (m *Manager) acknowledgeVm(ipAddr net.IP,
	authInfo *srpc.AuthInformation) error {
	vm, err := m.getVmAndLock(ipAddr)
	if err != nil {
		return err
	}
	defer vm.mutex.Unlock()
	if err := vm.checkAuth(authInfo); err != nil {
		return err
	}
	vm.destroyTimer.Stop()
	return nil
}

func (m *Manager) allocateVm(req proto.CreateVmRequest) (*vmInfoType, error) {
	address, err := m.getFreeAddress(req.SubnetId)
	if err != nil {
		return nil, err
	}
	freeAddress := true
	defer func() {
		if freeAddress {
			err := m.addAddressesToPool([]proto.Address{address}, true)
			if err != nil {
				m.Logger.Println(err)
			}
		}
	}()
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if err := m.checkSufficientCPUWithLock(req.MilliCPUs); err != nil {
		return nil, err
	}
	if err := m.checkSufficientMemoryWithLock(req.MemoryInMiB); err != nil {
		return nil, err
	}
	vm := &vmInfoType{
		VmInfo: proto.VmInfo{
			Address:       address,
			ImageName:     req.ImageName,
			ImageURL:      req.ImageURL,
			MemoryInMiB:   req.MemoryInMiB,
			MilliCPUs:     req.MilliCPUs,
			OwnerGroups:   req.OwnerGroups,
			SpreadVolumes: req.SpreadVolumes,
			State:         proto.StateStarting,
			Tags:          req.Tags,
		},
		manager: m,
		dirname: path.Join(m.StateDir, "VMs", address.IpAddress.String()),
		logger: prefixlogger.New(address.IpAddress.String()+": ",
			m.Logger),
	}
	m.vms[address.IpAddress.String()] = vm
	freeAddress = false
	return vm, nil
}

func (m *Manager) changeVmTags(ipAddr net.IP, authInfo *srpc.AuthInformation,
	tgs tags.Tags) error {
	vm, err := m.getVmAndLock(ipAddr)
	if err != nil {
		return err
	}
	defer vm.mutex.Unlock()
	if err := vm.checkAuth(authInfo); err != nil {
		return err
	}
	vm.Tags = tgs
	vm.setState(vm.State)
	return nil
}

func (m *Manager) checkVmHasHealthAgent(ipAddr net.IP) (bool, error) {
	vm, err := m.getVmAndLock(ipAddr)
	if err != nil {
		return false, err
	}
	defer vm.mutex.Unlock()
	if vm.State != proto.StateRunning {
		return false, nil
	}
	return vm.hasHealthAgent, nil
}

func (m *Manager) createVm(conn *srpc.Conn, decoder srpc.Decoder,
	encoder srpc.Encoder) error {

	sendError := func(conn *srpc.Conn, encoder srpc.Encoder, err error) error {
		return encoder.Encode(proto.CreateVmResponse{Error: err.Error()})
	}

	sendUpdate := func(conn *srpc.Conn, encoder srpc.Encoder,
		message string) error {
		response := proto.CreateVmResponse{ProgressMessage: message}
		if err := encoder.Encode(response); err != nil {
			return err
		}
		return conn.Flush()
	}

	var request proto.CreateVmRequest
	if err := decoder.Decode(&request); err != nil {
		return err
	}
	ownerUsers := make([]string, 1, len(request.OwnerUsers)+1)
	ownerUsers[0] = conn.Username()
	if ownerUsers[0] == "" {
		return errors.New("no authentication data")
	}
	ownerUsers = append(ownerUsers, request.OwnerUsers...)
	vm, err := m.allocateVm(request)
	if err != nil {
		if err := maybeDrainAll(conn, request); err != nil {
			return err
		}
		return sendError(conn, encoder, err)
	}
	defer func() {
		if vm == nil {
			return
		}
		m.mutex.Lock()
		delete(m.vms, vm.Address.IpAddress.String())
		err := m.addAddressesToPool([]proto.Address{vm.Address}, false)
		if err != nil {
			m.Logger.Println(err)
		}
		os.RemoveAll(vm.dirname)
		for _, volume := range vm.VolumeLocations {
			os.RemoveAll(path.Dir(volume.Filename))
		}
		m.mutex.Unlock()
	}()
	memoryError := tryAllocateMemory(request.MemoryInMiB)
	vm.OwnerUsers = ownerUsers
	vm.ownerUsers = make(map[string]struct{}, len(ownerUsers))
	for _, username := range ownerUsers {
		vm.ownerUsers[username] = struct{}{}
	}
	if err := os.MkdirAll(vm.dirname, dirPerms); err != nil {
		if err := maybeDrainAll(conn, request); err != nil {
			return err
		}
		return sendError(conn, encoder, err)
	}
	if request.ImageName != "" {
		if err := maybeDrainImage(conn, request.ImageDataSize); err != nil {
			return err
		}
		if err := sendUpdate(conn, encoder, "getting image"); err != nil {
			return err
		}
		client, err := srpc.DialHTTP("tcp", m.ImageServerAddress, 0)
		if err != nil {
			return sendError(conn, encoder,
				fmt.Errorf("error connecting to image server: %s: %s",
					m.ImageServerAddress, err))
		}
		defer client.Close()
		fs, err := getImage(client, request.ImageName, request.ImageTimeout)
		if err != nil {
			return sendError(conn, encoder, err)
		}
		objectClient := objclient.AttachObjectClient(client)
		defer objectClient.Close()
		size := computeSize(request.MinimumFreeBytes, request.RoundupPower,
			fs.EstimateUsage(0))
		if err := vm.setupVolumes(size, request); err != nil {
			return sendError(conn, encoder, err)
		}
		if err := sendUpdate(conn, encoder, "unpacking image"); err != nil {
			return err
		}
		err = util.WriteRaw(fs, objectClient, vm.VolumeLocations[0].Filename,
			privateFilePerms, mbr.TABLE_TYPE_MSDOS, request.MinimumFreeBytes,
			request.RoundupPower, true, true, m.Logger)
		if err != nil {
			return sendError(conn, encoder, err)
		}
		if fi, err := os.Stat(vm.VolumeLocations[0].Filename); err != nil {
			return sendError(conn, encoder, err)
		} else {
			vm.Volumes = []proto.Volume{{uint64(fi.Size())}}
		}
	} else if request.ImageDataSize > 0 {
		err := vm.copyRootVolume(request, conn, request.ImageDataSize)
		if err != nil {
			return err
		}
	} else if request.ImageURL != "" {
		if err := maybeDrainImage(conn, request.ImageDataSize); err != nil {
			return err
		}
		httpResponse, err := http.Get(request.ImageURL)
		if err != nil {
			return sendError(conn, encoder, err)
		}
		defer httpResponse.Body.Close()
		if httpResponse.StatusCode != http.StatusOK {
			return sendError(conn, encoder, errors.New(httpResponse.Status))
		}
		if httpResponse.ContentLength < 0 {
			return sendError(conn, encoder,
				errors.New("ContentLength from: "+request.ImageURL))
		}
		err = vm.copyRootVolume(request, httpResponse.Body,
			uint64(httpResponse.ContentLength))
		if err != nil {
			return sendError(conn, encoder, err)
		}
	} else {
		return sendError(conn, encoder, errors.New("no image specified"))
	}
	if request.UserDataSize > 0 {
		filename := path.Join(vm.dirname, "user-data.raw")
		err := copyData(filename, conn, request.UserDataSize, privateFilePerms)
		if err != nil {
			return sendError(conn, encoder, err)
		}
	}
	if len(request.SecondaryVolumes) > 0 {
		err := sendUpdate(conn, encoder, "creating secondary volumes")
		if err != nil {
			return err
		}
		for index, volume := range request.SecondaryVolumes {
			fname := vm.VolumeLocations[index+1].Filename
			cFlags := os.O_CREATE | os.O_TRUNC | os.O_RDWR
			file, err := os.OpenFile(fname, cFlags, privateFilePerms)
			if err != nil {
				return sendError(conn, encoder, err)
			} else {
				file.Close()
			}
			if err := setVolumeSize(fname, volume.Size); err != nil {
				return sendError(conn, encoder, err)
			}
		}
	}
	if len(memoryError) < 1 {
		msg := "waiting for test memory allocation"
		sendUpdate(conn, encoder, msg)
		vm.logger.Debugln(0, msg)
	}
	if err := <-memoryError; err != nil {
		return sendError(conn, encoder, err)
	}
	if err := sendUpdate(conn, encoder, "starting VM"); err != nil {
		return err
	}
	dhcpTimedOut, err := vm.startManaging(request.DhcpTimeout)
	if err != nil {
		return sendError(conn, encoder, err)
	}
	vm.destroyTimer = time.AfterFunc(time.Second*15, vm.autoDestroy)
	response := proto.CreateVmResponse{
		DhcpTimedOut: dhcpTimedOut,
		Final:        true,
		IpAddress:    vm.Address.IpAddress,
	}
	if err := encoder.Encode(response); err != nil {
		return err
	}
	vm = nil // Cancel cleanup.
	return nil
}

func (m *Manager) destroyVm(ipAddr net.IP,
	authInfo *srpc.AuthInformation) error {
	vm, err := m.getVmAndLock(ipAddr)
	if err != nil {
		return err
	}
	defer vm.mutex.Unlock()
	if err := vm.checkAuth(authInfo); err != nil {
		return err
	}
	switch vm.State {
	case proto.StateStarting:
		return errors.New("VM is starting")
	case proto.StateRunning:
		vm.setState(proto.StateDestroying)
		vm.commandChannel <- "quit"
	case proto.StateStopping:
		return errors.New("VM is stopping")
	case proto.StateStopped, proto.StateFailedToStart:
		vm.delete()
	case proto.StateDestroying:
		return errors.New("VM is already destroying")
	default:
		return errors.New("unknown state: " + vm.State.String())
	}
	return nil
}

func (m *Manager) getNumVMs() (uint, uint) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	var numRunning, numStopped uint
	for _, vm := range m.vms {
		if vm.State == proto.StateRunning {
			numRunning++
		} else {
			numStopped++
		}
	}
	return numRunning, numStopped
}

func (m *Manager) getVmAndLock(ipAddr net.IP) (*vmInfoType, error) {
	ipStr := ipAddr.String()
	m.mutex.RLock()
	if vm := m.vms[ipStr]; vm == nil {
		m.mutex.RUnlock()
		return nil, fmt.Errorf("no VM with IP address: %s found", ipStr)
	} else {
		vm.mutex.Lock()
		m.mutex.RUnlock()
		return vm, nil
	}
}

func (m *Manager) getVmBootLog(ipAddr net.IP) (io.ReadCloser, error) {
	vm, err := m.getVmAndLock(ipAddr)
	if err != nil {
		return nil, err
	}
	filename := path.Join(vm.dirname, "bootlog")
	vm.mutex.Unlock()
	return os.Open(filename)
}

func (m *Manager) getVmInfo(ipAddr net.IP) (proto.VmInfo, error) {
	vm, err := m.getVmAndLock(ipAddr)
	if err != nil {
		return proto.VmInfo{}, err
	}
	defer vm.mutex.Unlock()
	return vm.VmInfo, nil
}

func (m *Manager) getVmUserData(ipAddr net.IP) (io.ReadCloser, error) {
	vm, err := m.getVmAndLock(ipAddr)
	if err != nil {
		return nil, err
	}
	filename := path.Join(vm.dirname, "user-data.raw")
	vm.mutex.Unlock()
	return os.Open(filename)
}

func (m *Manager) listVMs(doSort bool) []string {
	m.mutex.RLock()
	ipAddrs := make([]string, 0, len(m.vms))
	for ipAddr := range m.vms {
		ipAddrs = append(ipAddrs, ipAddr)
	}
	m.mutex.RUnlock()
	if doSort {
		verstr.Sort(ipAddrs)
	}
	return ipAddrs
}

func (m *Manager) replaceVmImage(conn *srpc.Conn, decoder srpc.Decoder,
	encoder srpc.Encoder, authInfo *srpc.AuthInformation) error {

	sendError := func(conn *srpc.Conn, encoder srpc.Encoder, err error) error {
		return encoder.Encode(proto.ReplaceVmImageResponse{Error: err.Error()})
	}

	sendUpdate := func(conn *srpc.Conn, encoder srpc.Encoder,
		message string) error {
		response := proto.ReplaceVmImageResponse{ProgressMessage: message}
		if err := encoder.Encode(response); err != nil {
			return err
		}
		return conn.Flush()
	}

	var request proto.ReplaceVmImageRequest
	if err := decoder.Decode(&request); err != nil {
		return err
	}
	vm, err := m.getVmAndLock(request.IpAddress)
	if err != nil {
		return err
	}
	defer vm.mutex.Unlock()
	if err := vm.checkAuth(authInfo); err != nil {
		return err
	}
	if vm.State != proto.StateStopped {
		if err := maybeDrainImage(conn, request.ImageDataSize); err != nil {
			return err
		}
		return sendError(conn, encoder, errors.New("VM is not stopped"))
	}
	tmpRootFilename := vm.VolumeLocations[0].Filename + ".new"
	defer os.Remove(tmpRootFilename)
	var newSize uint64
	if request.ImageName != "" {
		if err := maybeDrainImage(conn, request.ImageDataSize); err != nil {
			return err
		}
		if err := sendUpdate(conn, encoder, "getting image"); err != nil {
			return err
		}
		client, err := srpc.DialHTTP("tcp", m.ImageServerAddress, 0)
		if err != nil {
			return sendError(conn, encoder,
				fmt.Errorf("error connecting to image server: %s: %s",
					m.ImageServerAddress, err))
		}
		defer client.Close()
		fs, err := getImage(client, request.ImageName, request.ImageTimeout)
		if err != nil {
			return sendError(conn, encoder, err)
		}
		objectClient := objclient.AttachObjectClient(client)
		defer objectClient.Close()
		if err := sendUpdate(conn, encoder, "unpacking image"); err != nil {
			return err
		}
		err = util.WriteRaw(fs, objectClient, tmpRootFilename, privateFilePerms,
			mbr.TABLE_TYPE_MSDOS, request.MinimumFreeBytes,
			request.RoundupPower, true, true, m.Logger)
		if err != nil {
			return sendError(conn, encoder, err)
		}
		if fi, err := os.Stat(tmpRootFilename); err != nil {
			return sendError(conn, encoder, err)
		} else {
			newSize = uint64(fi.Size())
		}
	} else if request.ImageDataSize > 0 {
		err := copyData(tmpRootFilename, conn, request.ImageDataSize,
			privateFilePerms)
		if err != nil {
			return err
		}
		newSize = computeSize(request.MinimumFreeBytes, request.RoundupPower,
			request.ImageDataSize)
		if err := setVolumeSize(tmpRootFilename, newSize); err != nil {
			return sendError(conn, encoder, err)
		}
	} else if request.ImageURL != "" {
		if err := maybeDrainImage(conn, request.ImageDataSize); err != nil {
			return err
		}
		httpResponse, err := http.Get(request.ImageURL)
		if err != nil {
			return sendError(conn, encoder, err)
		}
		defer httpResponse.Body.Close()
		if httpResponse.StatusCode != http.StatusOK {
			return sendError(conn, encoder, errors.New(httpResponse.Status))
		}
		if httpResponse.ContentLength < 0 {
			return sendError(conn, encoder,
				errors.New("ContentLength from: "+request.ImageURL))
		}
		err = copyData(tmpRootFilename, httpResponse.Body,
			uint64(httpResponse.ContentLength), privateFilePerms)
		if err != nil {
			return sendError(conn, encoder, err)
		}
		newSize = computeSize(request.MinimumFreeBytes, request.RoundupPower,
			uint64(httpResponse.ContentLength))
		if err := setVolumeSize(tmpRootFilename, newSize); err != nil {
			return sendError(conn, encoder, err)
		}
	} else {
		return sendError(conn, encoder, errors.New("no image specified"))
	}
	rootFilename := vm.VolumeLocations[0].Filename
	oldRootFilename := vm.VolumeLocations[0].Filename + ".old"
	if err := os.Rename(rootFilename, oldRootFilename); err != nil {
		return sendError(conn, encoder, err)
	}
	if err := os.Rename(tmpRootFilename, rootFilename); err != nil {
		os.Rename(oldRootFilename, rootFilename)
		return sendError(conn, encoder, err)
	}
	if request.ImageName != "" {
		vm.ImageName = request.ImageName
	}
	vm.Volumes[0].Size = newSize
	vm.setState(vm.State)
	response := proto.ReplaceVmImageResponse{
		Final: true,
	}
	if err := encoder.Encode(response); err != nil {
		return err
	}
	return nil
}

func (m *Manager) restoreVmImage(ipAddr net.IP,
	authInfo *srpc.AuthInformation) error {
	vm, err := m.getVmAndLock(ipAddr)
	if err != nil {
		return err
	}
	defer vm.mutex.Unlock()
	if err := vm.checkAuth(authInfo); err != nil {
		return err
	}
	if vm.State != proto.StateStopped {
		return errors.New("VM is not stopped")
	}
	rootFilename := vm.VolumeLocations[0].Filename
	oldRootFilename := vm.VolumeLocations[0].Filename + ".old"
	fi, err := os.Stat(oldRootFilename)
	if err != nil {
		return err
	}
	if err := os.Rename(oldRootFilename, rootFilename); err != nil {
		return err
	}
	vm.Volumes[0].Size = uint64(fi.Size())
	vm.setState(vm.State)
	return nil
}

func (m *Manager) startVm(ipAddr net.IP, authInfo *srpc.AuthInformation,
	dhcpTimeout time.Duration) (bool, error) {
	vm, err := m.getVmAndLock(ipAddr)
	if err != nil {
		return false, err
	}
	defer vm.mutex.Unlock()
	if err := vm.checkAuth(authInfo); err != nil {
		return false, err
	}
	if err := checkAvailableMemory(vm.MemoryInMiB); err != nil {
		return false, err
	}
	switch vm.State {
	case proto.StateStarting:
		return false, errors.New("VM is already starting")
	case proto.StateRunning:
		return false, errors.New("VM is running")
	case proto.StateStopping:
		return false, errors.New("VM is stopping")
	case proto.StateStopped, proto.StateFailedToStart:
		vm.setState(proto.StateStarting)
		return vm.startManaging(dhcpTimeout)
	case proto.StateDestroying:
		return false, errors.New("VM is destroying")
	default:
		return false, errors.New("unknown state: " + vm.State.String())
	}
	return false, nil
}

func (m *Manager) stopVm(ipAddr net.IP, authInfo *srpc.AuthInformation) error {
	vm, err := m.getVmAndLock(ipAddr)
	if err != nil {
		return err
	}
	defer vm.mutex.Unlock()
	if err := vm.checkAuth(authInfo); err != nil {
		return err
	}
	switch vm.State {
	case proto.StateStarting:
		return errors.New("VM is starting")
	case proto.StateRunning:
		vm.setState(proto.StateStopping)
		vm.commandChannel <- "system_powerdown"
		time.AfterFunc(time.Second*15, vm.kill)
	case proto.StateStopping:
		return errors.New("VM is stopping")
	case proto.StateStopped, proto.StateFailedToStart:
		return errors.New("VM is already stopped")
	case proto.StateDestroying:
		return errors.New("VM is destroying")
	default:
		return errors.New("unknown state: " + vm.State.String())
	}
	return nil
}

func (vm *vmInfoType) autoDestroy() {
	vm.logger.Println("VM was not acknowledged, destroying")
	authInfo := &srpc.AuthInformation{HaveMethodAccess: true}
	if err := vm.manager.destroyVm(vm.Address.IpAddress, authInfo); err != nil {
		vm.logger.Println(err)
	}
}

func (vm *vmInfoType) checkAuth(authInfo *srpc.AuthInformation) error {
	if authInfo.HaveMethodAccess {
		return nil
	}
	if _, ok := vm.ownerUsers[authInfo.Username]; ok {
		return nil
	}
	return errorNoAccessToResource
}

func (vm *vmInfoType) copyRootVolume(request proto.CreateVmRequest,
	reader io.Reader, dataSize uint64) error {
	size := computeSize(request.MinimumFreeBytes, request.RoundupPower,
		dataSize)
	if err := vm.setupVolumes(size, request); err != nil {
		return err
	}
	err := copyData(vm.VolumeLocations[0].Filename, reader, dataSize,
		privateFilePerms)
	if err != nil {
		return err
	}
	vm.Volumes = []proto.Volume{{size}}
	return setVolumeSize(vm.VolumeLocations[0].Filename, size)
}

func (vm *vmInfoType) delete() {
	vm.manager.DhcpServer.RemoveLease(vm.Address.IpAddress)
	vm.manager.mutex.Lock()
	delete(vm.manager.vms, vm.Address.IpAddress.String())
	err := vm.manager.addAddressesToPool([]proto.Address{vm.Address}, false)
	vm.manager.mutex.Unlock()
	if err != nil {
		vm.manager.Logger.Println(err)
	}
	for _, volume := range vm.VolumeLocations {
		os.Remove(volume.Filename)
		if volume.DirectoryToCleanup != "" {
			os.RemoveAll(volume.DirectoryToCleanup)
		}
	}
	os.RemoveAll(vm.dirname)
}

func (vm *vmInfoType) kill() {
	vm.mutex.Lock()
	defer vm.mutex.Unlock()
	if vm.State == proto.StateStopping {
		vm.commandChannel <- "quit"
	}
}

func (vm *vmInfoType) monitor(monitorSock net.Conn,
	commandChannel <-chan string) {
	vm.hasHealthAgent = false
	defer monitorSock.Close()
	go vm.processMonitorResponses(monitorSock)
	cancelChannel := make(chan struct{}, 1)
	go vm.probeHealthAgent(cancelChannel)
	for command := range commandChannel {
		_, err := fmt.Fprintf(monitorSock, `{"execute":"%s"}`, command)
		if err != nil {
			vm.logger.Println(err)
		} else {
			vm.logger.Debugf(0, "sent %s command", command)
		}
	}
	cancelChannel <- struct{}{}
}

func (vm *vmInfoType) probeHealthAgent(cancel <-chan struct{}) {
	stopTime := time.Now().Add(time.Minute)
	for time.Until(stopTime) > 0 {
		select {
		case <-cancel:
			return
		default:
		}
		sleepUntil := time.Now().Add(time.Second)
		conn, err := net.DialTimeout("tcp",
			vm.Address.IpAddress.String()+":6910", time.Second*5)
		if err == nil {
			conn.Close()
			vm.mutex.Lock()
			vm.hasHealthAgent = true
			vm.mutex.Unlock()
			return
		}
		time.Sleep(time.Until(sleepUntil))
	}
}

func (vm *vmInfoType) processMonitorResponses(monitorSock net.Conn) {
	io.Copy(ioutil.Discard, monitorSock) // Read all and drop.
	vm.mutex.Lock()
	defer vm.mutex.Unlock()
	switch vm.State {
	case proto.StateStarting:
		return
	case proto.StateRunning:
		return
	case proto.StateFailedToStart:
		return
	case proto.StateStopping:
		vm.setState(proto.StateStopped)
	case proto.StateStopped:
		return
	case proto.StateDestroying:
		vm.delete()
		return
	default:
		vm.logger.Println("unknown state: " + vm.State.String())
	}
	close(vm.commandChannel)
}

func (vm *vmInfoType) setState(state proto.State) {
	vm.State = state
	filename := path.Join(vm.dirname, "info.json")
	err := json.WriteToFile(filename, publicFilePerms, "    ", vm)
	if err != nil {
		vm.logger.Println(err)
		return
	}
}

func (vm *vmInfoType) setupVolumes(rootSize uint64,
	request proto.CreateVmRequest) error {
	volumeDirectories, err := vm.manager.getVolumeDirectories(rootSize,
		request.SecondaryVolumes, request.SpreadVolumes)
	if err != nil {
		return err
	}
	ipAddress := vm.Address.IpAddress.String()
	volumeDirectory := path.Join(volumeDirectories[0], ipAddress)
	os.RemoveAll(volumeDirectory)
	if err := os.MkdirAll(volumeDirectory, dirPerms); err != nil {
		return err
	}
	filename := path.Join(volumeDirectory, "root")
	vm.VolumeLocations = append(vm.VolumeLocations,
		volumeType{volumeDirectory, filename})
	for index := range request.SecondaryVolumes {
		volumeDirectory := path.Join(volumeDirectories[index+1], ipAddress)
		os.RemoveAll(volumeDirectory)
		if err := os.MkdirAll(volumeDirectory, dirPerms); err != nil {
			return err
		}
		filename := path.Join(volumeDirectory,
			fmt.Sprintf("secondary-volume.%d", index))
		vm.VolumeLocations = append(vm.VolumeLocations,
			volumeType{volumeDirectory, filename})
	}
	return nil
}

func (vm *vmInfoType) startManaging(dhcpTimeout time.Duration) (bool, error) {
	vm.monitorSockname = path.Join(vm.dirname, "monitor.sock")
	switch vm.State {
	case proto.StateStarting:
	case proto.StateRunning:
	case proto.StateFailedToStart:
	case proto.StateStopping:
		monitorSock, err := net.Dial("unix", vm.monitorSockname)
		if err == nil {
			commandChannel := make(chan string, 1)
			vm.commandChannel = commandChannel
			go vm.monitor(monitorSock, commandChannel)
			commandChannel <- "qmp_capabilities"
			vm.kill()
		}
		return false, nil
	case proto.StateStopped:
		return false, nil
	case proto.StateDestroying:
		vm.delete()
		return false, nil
	default:
		vm.logger.Println("unknown state: " + vm.State.String())
		return false, nil
	}
	vm.manager.DhcpServer.AddLease(vm.Address)
	monitorSock, err := net.Dial("unix", vm.monitorSockname)
	if err != nil {
		vm.logger.Debugf(0, "error connecting to: %s: %s\n",
			vm.monitorSockname, err)
		if err := vm.startVm(); err != nil {
			vm.logger.Println(err)
			vm.setState(proto.StateFailedToStart)
			return false, err
		}
		monitorSock, err = net.Dial("unix", vm.monitorSockname)
	}
	if err != nil {
		vm.logger.Println(err)
		vm.setState(proto.StateFailedToStart)
		return false, err
	}
	commandChannel := make(chan string, 1)
	vm.commandChannel = commandChannel
	go vm.monitor(monitorSock, commandChannel)
	commandChannel <- "qmp_capabilities"
	vm.setState(proto.StateRunning)
	if dhcpTimeout > 0 {
		ackChan := vm.manager.DhcpServer.MakeAcknowledgmentChannel(
			vm.Address.IpAddress)
		timer := time.NewTimer(dhcpTimeout)
		select {
		case <-ackChan:
			timer.Stop()
		case <-timer.C:
			return true, nil
		}
	}
	return false, nil
}

func (vm *vmInfoType) startVm() error {
	if err := checkAvailableMemory(vm.MemoryInMiB); err != nil {
		return err
	}
	nCpus := vm.MilliCPUs / 1000
	if nCpus < 1 {
		nCpus = 1
	}
	if nCpus*1000 < vm.MilliCPUs {
		nCpus++
	}
	bootlogFilename := path.Join(vm.dirname, "bootlog")
	cmd := exec.Command("qemu-system-x86_64", "-machine", "pc,accel=kvm",
		"-nodefaults",
		"-name", vm.Address.IpAddress.String(),
		"-m", fmt.Sprintf("%dM", vm.MemoryInMiB),
		"-smp", fmt.Sprintf("cpus=%d", nCpus),
		"-net", "nic,model=virtio,macaddr="+vm.Address.MacAddress,
		"-net", "tap",
		"-serial", "file:"+bootlogFilename,
		"-chroot", "/tmp",
		"-runas", vm.manager.Username,
		"-qmp", "unix:"+vm.monitorSockname+",server,nowait",
		"-daemonize")
	if vm.manager.ShowVgaConsole {
		cmd.Args = append(cmd.Args, "-vga", "std")
	} else {
		cmd.Args = append(cmd.Args, "-nographic")
	}
	for _, volume := range vm.VolumeLocations {
		cmd.Args = append(cmd.Args,
			"-drive", "file="+volume.Filename+",format=raw")
	}
	os.Remove(bootlogFilename)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error starting QEMU: %s: %s", err, output)
	}
	return nil
}
