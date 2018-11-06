package manager

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/gob"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path"
	"time"

	hyperclient "github.com/Symantec/Dominator/hypervisor/client"
	imclient "github.com/Symantec/Dominator/imageserver/client"
	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/filesystem/util"
	"github.com/Symantec/Dominator/lib/fsutil"
	"github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/lib/log/prefixlogger"
	"github.com/Symantec/Dominator/lib/mbr"
	libnet "github.com/Symantec/Dominator/lib/net"
	objclient "github.com/Symantec/Dominator/lib/objectserver/client"
	"github.com/Symantec/Dominator/lib/rsync"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/lib/tags"
	"github.com/Symantec/Dominator/lib/verstr"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
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

func createTapDevice(bridge string) (*os.File, error) {
	tapFile, tapName, err := libnet.CreateTapDevice()
	if err != nil {
		return nil, fmt.Errorf("error creating tap device: %s", err)
	}
	doAutoClose := true
	defer func() {
		if doAutoClose {
			tapFile.Close()
		}
	}()
	cmd := exec.Command("ip", "link", "set", tapName, "up")
	if output, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("error upping: %s: %s", err, output)
	}
	cmd = exec.Command("ip", "link", "set", tapName, "master", bridge)
	if output, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("error attaching: %s: %s", err, output)
	}
	doAutoClose = false
	return tapFile, nil
}

func getImage(client *srpc.Client, searchName string,
	imageTimeout time.Duration) (*filesystem.FileSystem, string, error) {
	if isDir, err := imclient.CheckDirectory(client, searchName); err != nil {
		return nil, "", err
	} else if isDir {
		imageName, err := imclient.FindLatestImage(client, searchName, false)
		if err != nil {
			return nil, "", err
		}
		if imageName == "" {
			return nil, "", errors.New("no images in directory: " + searchName)
		}
		img, err := imclient.GetImage(client, imageName)
		if err != nil {
			return nil, "", err
		}
		img.FileSystem.RebuildInodePointers()
		return img.FileSystem, imageName, nil
	}
	img, err := imclient.GetImageWithTimeout(client, searchName, imageTimeout)
	if err != nil {
		return nil, "", err
	}
	if img == nil {
		return nil, "", errors.New("timeout getting image")
	}
	img.FileSystem.RebuildInodePointers()
	return img.FileSystem, searchName, nil
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
	vm, err := m.getVmLockAndAuth(ipAddr, authInfo, nil)
	if err != nil {
		return err
	}
	defer vm.mutex.Unlock()
	vm.destroyTimer.Stop()
	return nil
}

func (m *Manager) allocateVm(req proto.CreateVmRequest,
	authInfo *srpc.AuthInformation) (*vmInfoType, error) {
	address, subnetId, err := m.getFreeAddress(req.SubnetId, authInfo)
	if err != nil {
		return nil, err
	}
	freeAddress := true
	defer func() {
		if freeAddress {
			err := m.releaseAddressInPool(address)
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
	var ipAddress string
	if len(address.IpAddress) < 1 {
		ipAddress = "0.0.0.0"
	} else {
		ipAddress = address.IpAddress.String()
	}
	vm := &vmInfoType{
		VmInfo: proto.VmInfo{
			Address:       address,
			Hostname:      req.Hostname,
			ImageName:     req.ImageName,
			ImageURL:      req.ImageURL,
			MemoryInMiB:   req.MemoryInMiB,
			MilliCPUs:     req.MilliCPUs,
			OwnerGroups:   req.OwnerGroups,
			SpreadVolumes: req.SpreadVolumes,
			State:         proto.StateStarting,
			Tags:          req.Tags,
			SubnetId:      subnetId,
		},
		manager:          m,
		dirname:          path.Join(m.StateDir, "VMs", ipAddress),
		ipAddress:        ipAddress,
		logger:           prefixlogger.New(ipAddress+": ", m.Logger),
		metadataChannels: make(map[chan<- string]struct{}),
	}
	m.vms[ipAddress] = vm
	freeAddress = false
	return vm, nil
}

func (m *Manager) becomePrimaryVmOwner(ipAddr net.IP,
	authInfo *srpc.AuthInformation) error {
	vm, err := m.getVmLockAndAuth(ipAddr, authInfo, nil)
	if err != nil {
		return err
	}
	defer vm.mutex.Unlock()
	if vm.OwnerUsers[0] == authInfo.Username {
		return errors.New("you already are the primary owner")
	}
	ownerUsers := make([]string, 1, len(vm.OwnerUsers[0]))
	ownerUsers[0] = authInfo.Username
	for _, user := range vm.OwnerUsers {
		if user != authInfo.Username {
			ownerUsers = append(ownerUsers, user)
		}
	}
	vm.OwnerUsers = ownerUsers
	vm.ownerUsers = make(map[string]struct{}, len(ownerUsers))
	for _, user := range ownerUsers {
		vm.ownerUsers[user] = struct{}{}
	}
	vm.writeAndSendInfo()
	return nil
}

func (m *Manager) changeVmOwnerUsers(ipAddr net.IP,
	authInfo *srpc.AuthInformation, extraUsers []string) error {
	vm, err := m.getVmLockAndAuth(ipAddr, authInfo, nil)
	if err != nil {
		return err
	}
	defer vm.mutex.Unlock()
	ownerUsers := make([]string, 1, len(extraUsers)+1)
	ownerUsers[0] = vm.OwnerUsers[0]
	for _, user := range extraUsers {
		ownerUsers = append(ownerUsers, user)
	}
	vm.OwnerUsers = ownerUsers
	vm.ownerUsers = make(map[string]struct{}, len(ownerUsers))
	for _, user := range ownerUsers {
		vm.ownerUsers[user] = struct{}{}
	}
	vm.writeAndSendInfo()
	return nil
}

func (m *Manager) changeVmTags(ipAddr net.IP, authInfo *srpc.AuthInformation,
	tgs tags.Tags) error {
	vm, err := m.getVmLockAndAuth(ipAddr, authInfo, nil)
	if err != nil {
		return err
	}
	defer vm.mutex.Unlock()
	vm.Tags = tgs
	vm.writeAndSendInfo()
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

func (m *Manager) commitImportedVm(ipAddr net.IP,
	authInfo *srpc.AuthInformation) error {
	vm, err := m.getVmLockAndAuth(ipAddr, authInfo, nil)
	if err != nil {
		return err
	}
	defer vm.mutex.Unlock()
	if !vm.Uncommitted {
		return fmt.Errorf("%s is already committed")
	}
	if err := m.registerAddress(vm.Address); err != nil {
		return err
	}
	vm.Uncommitted = false
	vm.writeAndSendInfo()
	return nil
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
	vm, err := m.allocateVm(request, conn.GetAuthInformation())
	if err != nil {
		if err := maybeDrainAll(conn, request); err != nil {
			return err
		}
		return sendError(conn, encoder, err)
	}
	defer func() {
		vm.cleanup() // Evaluate vm at return time, not defer time.
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
		fs, imageName, err := getImage(client, request.ImageName,
			request.ImageTimeout)
		if err != nil {
			return sendError(conn, encoder, err)
		}
		vm.ImageName = imageName
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
			vm.Volumes = []proto.Volume{{Size: uint64(fi.Size())}}
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
			vm.Volumes = append(vm.Volumes, volume)
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
	dhcpTimedOut, err := vm.startManaging(request.DhcpTimeout, false)
	if err != nil {
		return sendError(conn, encoder, err)
	}
	vm.destroyTimer = time.AfterFunc(time.Second*15, vm.autoDestroy)
	response := proto.CreateVmResponse{
		DhcpTimedOut: dhcpTimedOut,
		Final:        true,
		IpAddress:    net.ParseIP(vm.ipAddress),
	}
	if err := encoder.Encode(response); err != nil {
		return err
	}
	vm = nil // Cancel cleanup.
	return nil
}

func (m *Manager) deleteVmVolume(ipAddr net.IP, authInfo *srpc.AuthInformation,
	accessToken []byte, volumeIndex uint) error {
	if volumeIndex < 1 {
		return errors.New("cannot delete root volume")
	}
	vm, err := m.getVmLockAndAuth(ipAddr, authInfo, accessToken)
	if err != nil {
		return err
	}
	defer vm.mutex.Unlock()
	if volumeIndex >= uint(len(vm.VolumeLocations)) {
		return errors.New("volume index too large")
	}
	if vm.State != proto.StateStopped {
		return errors.New("VM is not stopped")
	}
	if err := os.Remove(vm.VolumeLocations[volumeIndex].Filename); err != nil {
		return err
	}
	os.Remove(vm.VolumeLocations[volumeIndex].DirectoryToCleanup)
	volumeLocations := make([]volumeType, 0, len(vm.VolumeLocations)-1)
	volumes := make([]proto.Volume, 0, len(vm.VolumeLocations)-1)
	for index, volume := range vm.VolumeLocations {
		if uint(index) != volumeIndex {
			volumeLocations = append(volumeLocations, volume)
			volumes = append(volumes, vm.Volumes[index])
		}
	}
	vm.VolumeLocations = volumeLocations
	vm.Volumes = volumes
	vm.writeAndSendInfo()
	return nil
}

func (m *Manager) destroyVm(ipAddr net.IP, authInfo *srpc.AuthInformation,
	accessToken []byte) error {
	vm, err := m.getVmLockAndAuth(ipAddr, authInfo, accessToken)
	if err != nil {
		return err
	}
	defer vm.mutex.Unlock()
	switch vm.State {
	case proto.StateStarting:
		return errors.New("VM is starting")
	case proto.StateRunning:
		vm.setState(proto.StateDestroying)
		vm.commandChannel <- "quit"
	case proto.StateStopping:
		return errors.New("VM is stopping")
	case proto.StateStopped, proto.StateFailedToStart, proto.StateMigrating:
		vm.delete()
	case proto.StateDestroying:
		return errors.New("VM is already destroying")
	default:
		return errors.New("unknown state: " + vm.State.String())
	}
	return nil
}

func (m *Manager) discardVmAccessToken(ipAddr net.IP,
	authInfo *srpc.AuthInformation, accessToken []byte) error {
	vm, err := m.getVmLockAndAuth(ipAddr, authInfo, accessToken)
	if err != nil {
		return err
	}
	defer vm.mutex.Unlock()
	for index := range vm.accessToken { // Scrub token.
		vm.accessToken[index] = 0
	}
	vm.accessToken = nil
	return nil
}

func (m *Manager) discardVmOldImage(ipAddr net.IP,
	authInfo *srpc.AuthInformation) error {
	vm, err := m.getVmLockAndAuth(ipAddr, authInfo, nil)
	if err != nil {
		return err
	}
	defer vm.mutex.Unlock()
	return os.Remove(vm.VolumeLocations[0].Filename + ".old")
}

func (m *Manager) discardVmOldUserData(ipAddr net.IP,
	authInfo *srpc.AuthInformation) error {
	vm, err := m.getVmLockAndAuth(ipAddr, authInfo, nil)
	if err != nil {
		return err
	}
	defer vm.mutex.Unlock()
	return os.Remove(path.Join(vm.dirname, "user-data.old"))
}

func (m *Manager) discardVmSnapshot(ipAddr net.IP,
	authInfo *srpc.AuthInformation) error {
	vm, err := m.getVmLockAndAuth(ipAddr, authInfo, nil)
	if err != nil {
		return err
	}
	defer vm.mutex.Unlock()
	return vm.discardSnapshot()
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

func (m *Manager) getVmAccessToken(ipAddr net.IP,
	authInfo *srpc.AuthInformation, lifetime time.Duration) ([]byte, error) {
	if lifetime < time.Minute {
		return nil, errors.New("lifetime is less than 1 minute")
	}
	if lifetime > time.Hour*24 {
		return nil, errors.New("lifetime is greater than 1 day")
	}
	vm, err := m.getVmLockAndAuth(ipAddr, authInfo, nil)
	if err != nil {
		return nil, err
	}
	defer vm.mutex.Unlock()
	if vm.accessToken != nil {
		return nil, errors.New("someone else has the access token")
	}
	vm.accessToken = nil
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		return nil, err
	}
	vm.accessToken = token
	cleanupNotifier := make(chan struct{}, 1)
	vm.accessTokenCleanupNotifier = cleanupNotifier
	go func() {
		timer := time.NewTimer(lifetime)
		select {
		case <-timer.C:
		case <-cleanupNotifier:
		}
		vm.mutex.Lock()
		defer vm.mutex.Unlock()
		for index := 0; index < len(vm.accessToken); index++ {
			vm.accessToken[index] = 0 // Scrub sensitive data.
		}
		vm.accessToken = nil
	}()
	return token, nil
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

func (m *Manager) getVmLockAndAuth(ipAddr net.IP,
	authInfo *srpc.AuthInformation, accessToken []byte) (*vmInfoType, error) {
	vm, err := m.getVmAndLock(ipAddr)
	if err != nil {
		return nil, err
	}
	if err := vm.checkAuth(authInfo, accessToken); err != nil {
		vm.mutex.Unlock()
		return nil, err
	}
	return vm, nil
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

func (m *Manager) getVmUserData(ipAddr net.IP, authInfo *srpc.AuthInformation,
	accessToken []byte) (io.ReadCloser, uint64, error) {
	vm, err := m.getVmLockAndAuth(ipAddr, authInfo, accessToken)
	if err != nil {
		return nil, 0, err
	}
	filename := path.Join(vm.dirname, "user-data.raw")
	vm.mutex.Unlock()
	if file, err := os.Open(filename); err != nil {
		return nil, 0, err
	} else if fi, err := file.Stat(); err != nil {
		return nil, 0, err
	} else {
		return file, uint64(fi.Size()), nil
	}
}

func (m *Manager) getVmVolume(conn *srpc.Conn, decoder srpc.Decoder,
	encoder srpc.Encoder) error {
	var request proto.GetVmVolumeRequest
	if err := decoder.Decode(&request); err != nil {
		return err
	}
	vm, err := m.getVmLockAndAuth(request.IpAddress, conn.GetAuthInformation(),
		request.AccessToken)
	if err != nil {
		return encoder.Encode(proto.GetVmVolumeResponse{Error: err.Error()})
	}
	defer vm.mutex.Unlock()
	if request.VolumeIndex >= uint(len(vm.VolumeLocations)) {
		return encoder.Encode(proto.GetVmVolumeResponse{
			Error: "index too large"})
	}
	file, err := os.Open(vm.VolumeLocations[request.VolumeIndex].Filename)
	if err != nil {
		return encoder.Encode(proto.GetVmVolumeResponse{Error: err.Error()})
	}
	defer file.Close()
	if err := encoder.Encode(proto.GetVmVolumeResponse{}); err != nil {
		return err
	}
	if err := conn.Flush(); err != nil {
		return err
	}
	return rsync.ServeBlocks(conn, decoder, encoder, file,
		vm.Volumes[request.VolumeIndex].Size)
}

func (m *Manager) importLocalVm(authInfo *srpc.AuthInformation,
	request proto.ImportLocalVmRequest) error {
	if !bytes.Equal(m.importCookie, request.VerificationCookie) {
		return fmt.Errorf("bad verification cookie: you are not root")
	}
	request.VmInfo.OwnerUsers = []string{authInfo.Username}
	request.VmInfo.Uncommitted = true
	volumeDirectories := make(map[string]struct{}, len(m.volumeDirectories))
	for _, dirname := range m.volumeDirectories {
		volumeDirectories[dirname] = struct{}{}
	}
	volumes := make([]proto.Volume, 0, len(request.VolumeFilenames))
	for index, filename := range request.VolumeFilenames {
		dirname := path.Dir(path.Dir(path.Dir(filename)))
		if _, ok := volumeDirectories[dirname]; !ok {
			return fmt.Errorf("%s not in a volume directory", filename)
		}
		if fi, err := os.Lstat(filename); err != nil {
			return err
		} else if fi.Mode()&os.ModeType != 0 {
			return fmt.Errorf("%s is not a regular file", filename)
		} else {
			var volumeFormat proto.VolumeFormat
			if index < len(request.VmInfo.Volumes) {
				volumeFormat = request.VmInfo.Volumes[index].Format
			}
			volumes = append(volumes, proto.Volume{
				Size:   uint64(fi.Size()),
				Format: volumeFormat,
			})
		}
	}
	request.Volumes = volumes
	if err := <-tryAllocateMemory(request.MemoryInMiB); err != nil {
		return err
	}
	ipAddress := request.Address.IpAddress.String()
	vm := &vmInfoType{
		VmInfo:           request.VmInfo,
		manager:          m,
		dirname:          path.Join(m.StateDir, "VMs", ipAddress),
		ipAddress:        ipAddress,
		ownerUsers:       map[string]struct{}{authInfo.Username: struct{}{}},
		logger:           prefixlogger.New(ipAddress+": ", m.Logger),
		metadataChannels: make(map[chan<- string]struct{}),
	}
	vm.VmInfo.State = proto.StateStarting
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if _, ok := m.vms[ipAddress]; ok {
		return fmt.Errorf("%s already exists", ipAddress)
	}
	for _, poolAddress := range m.addressPool.Registered {
		if poolAddress.IpAddress.Equal(request.Address.IpAddress) ||
			poolAddress.MacAddress == request.Address.MacAddress {
			return fmt.Errorf("%s is in address pool", ipAddress)
		}
	}
	subnetId := m.getMatchingSubnet(request.Address.IpAddress)
	if subnetId == "" {
		return fmt.Errorf("no matching subnet for: %s\n", ipAddress)
	}
	vm.VmInfo.SubnetId = subnetId
	defer func() {
		if vm == nil {
			return
		}
		delete(m.vms, vm.ipAddress)
		m.sendVmInfo(vm.ipAddress, nil)
		os.RemoveAll(vm.dirname)
		for _, volume := range vm.VolumeLocations {
			os.RemoveAll(volume.DirectoryToCleanup)
		}
	}()
	if err := os.MkdirAll(vm.dirname, dirPerms); err != nil {
		return err
	}
	for index, sourceFilename := range request.VolumeFilenames {
		dirname := path.Join(path.Dir(path.Dir(path.Dir(sourceFilename))),
			ipAddress)
		if err := os.MkdirAll(dirname, dirPerms); err != nil {
			return err
		}
		var destFilename string
		if index == 0 {
			destFilename = path.Join(dirname, "root")
		} else {
			destFilename = path.Join(dirname,
				fmt.Sprintf("secondary-volume.%d", index-1))
		}
		if err := os.Link(sourceFilename, destFilename); err != nil {
			return err
		}
		vm.VolumeLocations = append(vm.VolumeLocations, volumeType{
			dirname, destFilename})
	}
	m.vms[ipAddress] = vm
	if _, err := vm.startManaging(0, true); err != nil {
		return err
	}
	vm = nil // Cancel cleanup.
	return nil
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

func (m *Manager) migrateVm(conn *srpc.Conn, decoder srpc.Decoder,
	encoder srpc.Encoder) error {
	var request proto.MigrateVmRequest
	if err := decoder.Decode(&request); err != nil {
		return err
	}
	hypervisor, err := srpc.DialHTTP("tcp", request.SourceHypervisor, 0)
	if err != nil {
		return err
	}
	defer hypervisor.Close()
	defer func() {
		req := proto.DiscardVmAccessTokenRequest{
			AccessToken: request.AccessToken,
			IpAddress:   request.IpAddress}
		var reply proto.DiscardVmAccessTokenResponse
		hypervisor.RequestReply("Hypervisor.DiscardVmAccessToken",
			req, &reply)
	}()
	ipAddress := request.IpAddress.String()
	m.mutex.RLock()
	_, ok := m.vms[ipAddress]
	subnetId := m.getMatchingSubnet(request.IpAddress)
	m.mutex.RUnlock()
	if ok {
		return errors.New("cannot migrate to the same hypervisor")
	}
	if subnetId == "" {
		return fmt.Errorf("no matching subnet for: %s\n", request.IpAddress)
	}
	getInfoRequest := proto.GetVmInfoRequest{request.IpAddress}
	var getInfoReply proto.GetVmInfoResponse
	err = hypervisor.RequestReply("Hypervisor.GetVmInfo", getInfoRequest,
		&getInfoReply)
	if err != nil {
		return err
	}
	accessToken := request.AccessToken
	vmInfo := getInfoReply.VmInfo
	if subnetId != vmInfo.SubnetId {
		return fmt.Errorf("subnet ID changing from: %s to: %s",
			vmInfo.SubnetId, subnetId)
	}
	if !request.IpAddress.Equal(vmInfo.Address.IpAddress) {
		return fmt.Errorf("inconsistent IP address: %s",
			vmInfo.Address.IpAddress)
	}
	if err := m.migrateVmChecks(vmInfo); err != nil {
		return err
	}
	volumeDirectories, err := m.getVolumeDirectories(vmInfo.Volumes[0].Size,
		vmInfo.Volumes[1:], vmInfo.SpreadVolumes)
	if err != nil {
		return err
	}
	vm := &vmInfoType{
		VmInfo:           vmInfo,
		VolumeLocations:  make([]volumeType, 0, len(volumeDirectories)),
		manager:          m,
		dirname:          path.Join(m.StateDir, "VMs", ipAddress),
		doNotWriteOrSend: true,
		ipAddress:        ipAddress,
		logger:           prefixlogger.New(ipAddress+": ", m.Logger),
		metadataChannels: make(map[chan<- string]struct{}),
	}
	vm.Uncommitted = true
	defer func() { // Evaluate vm at return time, not defer time.
		if vm == nil {
			return
		}
		vm.cleanup()
		hyperclient.PrepareVmForMigration(hypervisor, request.IpAddress,
			accessToken, false)
		if vmInfo.State == proto.StateRunning {
			hyperclient.StartVm(hypervisor, request.IpAddress, accessToken)
		}
	}()
	vm.ownerUsers = make(map[string]struct{}, len(vm.OwnerUsers))
	for _, username := range vm.OwnerUsers {
		vm.ownerUsers[username] = struct{}{}
	}
	if err := os.MkdirAll(vm.dirname, dirPerms); err != nil {
		return err
	}
	for index, _dirname := range volumeDirectories {
		dirname := path.Join(_dirname, ipAddress)
		if err := os.MkdirAll(dirname, dirPerms); err != nil {
			return err
		}
		var filename string
		if index == 0 {
			filename = path.Join(dirname, "root")
		} else {
			filename = path.Join(dirname,
				fmt.Sprintf("secondary-volume.%d", index-1))
		}
		vm.VolumeLocations = append(vm.VolumeLocations, volumeType{
			DirectoryToCleanup: dirname,
			Filename:           filename,
		})
	}
	if vmInfo.State == proto.StateStopped {
		err := hyperclient.PrepareVmForMigration(hypervisor, request.IpAddress,
			request.AccessToken, true)
		if err != nil {
			return err
		}
	}
	// Begin copying over the volumes.
	err = sendVmMigrationMessage(conn, encoder, "initial volume(s) copy")
	if err != nil {
		return err
	}
	if err := vm.migrateVmVolumes(hypervisor, accessToken); err != nil {
		return err
	}
	if vmInfo.State != proto.StateStopped {
		err = sendVmMigrationMessage(conn, encoder, "stopping VM")
		if err != nil {
			return err
		}
		err := hyperclient.StopVm(hypervisor, request.IpAddress,
			request.AccessToken)
		if err != nil {
			return err
		}
		err = hyperclient.PrepareVmForMigration(hypervisor, request.IpAddress,
			request.AccessToken, true)
		if err != nil {
			return err
		}
		err = sendVmMigrationMessage(conn, encoder, "update volume(s)")
		if err != nil {
			return err
		}
		if err := vm.migrateVmVolumes(hypervisor, accessToken); err != nil {
			return err
		}
	}
	err = migratevmUserData(hypervisor, path.Join(vm.dirname, "user-data.raw"),
		request.IpAddress, accessToken)
	if err != nil {
		return err
	}
	if err := sendVmMigrationMessage(conn, encoder, "starting VM"); err != nil {
		return err
	}
	m.mutex.Lock()
	m.vms[ipAddress] = vm
	m.mutex.Unlock()
	dhcpTimedOut, err := vm.startManaging(request.DhcpTimeout, false)
	if err != nil {
		return err
	}
	if dhcpTimedOut {
		return fmt.Errorf("DHCP timed out")
	}
	err = encoder.Encode(proto.MigrateVmResponse{RequestCommit: true})
	if err != nil {
		return err
	}
	if err := conn.Flush(); err != nil {
		return err
	}
	var reply proto.MigrateVmResponseResponse
	if err := decoder.Decode(&reply); err != nil {
		return err
	}
	if !reply.Commit {
		return fmt.Errorf("VM migration abandoned")
	}
	if err := m.registerAddress(vm.Address); err != nil {
		return err
	}
	vm.doNotWriteOrSend = false
	vm.Uncommitted = false
	vm.writeAndSendInfo()
	err = hyperclient.DestroyVm(hypervisor, request.IpAddress, accessToken)
	if err != nil {
		m.Logger.Printf("error cleaning up old migrated VM: %s\n", ipAddress)
	}
	vm = nil // Cancel cleanup.
	return nil
}

func sendVmMigrationMessage(conn *srpc.Conn, encoder srpc.Encoder,
	message string) error {
	request := proto.MigrateVmResponse{ProgressMessage: message}
	if err := encoder.Encode(request); err != nil {
		return err
	}
	return conn.Flush()
}

func (m *Manager) migrateVmChecks(vmInfo proto.VmInfo) error {
	switch vmInfo.State {
	case proto.StateStopped:
	case proto.StateRunning:
	default:
		return fmt.Errorf("VM state: %s is not stopped/running", vmInfo.State)
	}
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	if err := m.checkSufficientCPUWithLock(vmInfo.MilliCPUs); err != nil {
		return err
	}
	if err := m.checkSufficientMemoryWithLock(vmInfo.MemoryInMiB); err != nil {
		return err
	}
	if err := <-tryAllocateMemory(vmInfo.MemoryInMiB); err != nil {
		return err
	}
	return nil
}

func migratevmUserData(hypervisor *srpc.Client, filename string,
	ipAddr net.IP, accessToken []byte) error {
	conn, err := hypervisor.Call("Hypervisor.GetVmUserData")
	if err != nil {
		return err
	}
	defer conn.Close()
	encoder := gob.NewEncoder(conn)
	decoder := gob.NewDecoder(conn)
	request := proto.GetVmUserDataRequest{
		AccessToken: accessToken,
		IpAddress:   ipAddr,
	}
	if err := encoder.Encode(request); err != nil {
		return fmt.Errorf("error encoding request: %s", err)
	}
	if err := conn.Flush(); err != nil {
		return err
	}
	var reply proto.GetVmUserDataResponse
	if err := decoder.Decode(&reply); err != nil {
		return err
	}
	if err := errors.New(reply.Error); err != nil {
		return err
	}
	if reply.Length < 1 {
		return nil
	}
	writer, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_EXCL,
		privateFilePerms)
	if err != nil {
		io.CopyN(ioutil.Discard, conn, int64(reply.Length))
		return err
	}
	defer writer.Close()
	if _, err := io.CopyN(writer, conn, int64(reply.Length)); err != nil {
		return err
	}
	return nil
}

func (vm *vmInfoType) migrateVmVolumes(hypervisor *srpc.Client,
	accessToken []byte) error {
	for index, volume := range vm.VolumeLocations {
		_, err := migrateVmVolume(hypervisor, volume.Filename, uint(index),
			vm.Volumes[index].Size, vm.Address.IpAddress, accessToken)
		if err != nil {
			return err
		}
	}
	return nil
}

func migrateVmVolume(hypervisor *srpc.Client, filename string,
	volumeIndex uint, size uint64, ipAddr net.IP, accessToken []byte) (
	*rsync.Stats, error) {
	var initialFileSize uint64
	reader, err := os.OpenFile(filename, os.O_RDONLY, 0)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	} else {
		defer reader.Close()
		if fi, err := reader.Stat(); err != nil {
			return nil, err
		} else {
			initialFileSize = uint64(fi.Size())
			if initialFileSize > size {
				return nil, errors.New("file larger than volume")
			}
		}
	}
	writer, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE,
		privateFilePerms)
	if err != nil {
		return nil, err
	}
	defer writer.Close()
	request := proto.GetVmVolumeRequest{
		AccessToken: accessToken,
		IpAddress:   ipAddr,
		VolumeIndex: volumeIndex,
	}
	conn, err := hypervisor.Call("Hypervisor.GetVmVolume")
	if err != nil {
		if reader == nil {
			os.Remove(filename)
		}
		return nil, err
	}
	defer conn.Close()
	encoder := gob.NewEncoder(conn)
	decoder := gob.NewDecoder(conn)
	if err := encoder.Encode(request); err != nil {
		return nil, fmt.Errorf("error encoding request: %s", err)
	}
	if err := conn.Flush(); err != nil {
		return nil, err
	}
	var response proto.GetVmVolumeResponse
	if err := decoder.Decode(&response); err != nil {
		return nil, err
	}
	if err := errors.New(response.Error); err != nil {
		return nil, err
	}
	stats, err := rsync.GetBlocks(conn, decoder, encoder, reader, writer, size,
		initialFileSize)
	return &stats, err
}

func (m *Manager) notifyVmMetadataRequest(ipAddr net.IP, path string) {
	addr := ipAddr.String()
	m.mutex.RLock()
	vm, ok := m.vms[addr]
	m.mutex.RUnlock()
	if !ok {
		return
	}
	vm.mutex.Lock()
	defer vm.mutex.Unlock()
	for ch := range vm.metadataChannels {
		select {
		case ch <- path:
		default:
		}
	}
}

func (m *Manager) prepareVmForMigration(ipAddr net.IP,
	authInfoP *srpc.AuthInformation, accessToken []byte, enable bool) error {
	authInfo := *authInfoP
	authInfo.HaveMethodAccess = false // Require VM ownership or token.
	vm, err := m.getVmLockAndAuth(ipAddr, &authInfo, accessToken)
	if err != nil {
		return nil
	}
	defer vm.mutex.Unlock()
	if enable {
		if vm.Uncommitted {
			return errors.New("VM is uncommitted")
		}
		if vm.State != proto.StateStopped {
			return errors.New("VM is not stopped")
		}
		// Block reallocation of address until VM is destroyed, then release
		// claim on address.
		vm.Uncommitted = true
		vm.setState(proto.StateMigrating)
		if err := m.unregisterAddress(vm.Address); err != nil {
			vm.Uncommitted = false
			vm.setState(proto.StateStopped)
			return err
		}
	} else {
		if vm.State != proto.StateMigrating {
			return errors.New("VM is not migrating")
		}
		// Reclaim address and then allow reallocation if VM is later destroyed.
		if err := m.registerAddress(vm.Address); err != nil {
			vm.setState(proto.StateStopped)
			return err
		}
		vm.Uncommitted = false
		vm.setState(proto.StateStopped)
	}
	return nil
}

func (m *Manager) registerVmMetadataNotifier(ipAddr net.IP,
	authInfo *srpc.AuthInformation, pathChannel chan<- string) error {
	vm, err := m.getVmLockAndAuth(ipAddr, authInfo, nil)
	if err != nil {
		return err
	}
	defer vm.mutex.Unlock()
	vm.metadataChannels[pathChannel] = struct{}{}
	return nil
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
	vm, err := m.getVmLockAndAuth(request.IpAddress, authInfo, nil)
	if err != nil {
		return err
	}
	defer vm.mutex.Unlock()
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
		fs, imageName, err := getImage(client, request.ImageName,
			request.ImageTimeout)
		if err != nil {
			return sendError(conn, encoder, err)
		}
		request.ImageName = imageName
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
	vm.writeAndSendInfo()
	response := proto.ReplaceVmImageResponse{
		Final: true,
	}
	if err := encoder.Encode(response); err != nil {
		return err
	}
	return nil
}

func (m *Manager) replaceVmUserData(ipAddr net.IP, reader io.Reader,
	size uint64, authInfo *srpc.AuthInformation) error {
	vm, err := m.getVmLockAndAuth(ipAddr, authInfo, nil)
	if err != nil {
		return err
	}
	defer vm.mutex.Unlock()
	filename := path.Join(vm.dirname, "user-data.raw")
	oldFilename := filename + ".old"
	newFilename := filename + ".new"
	err = fsutil.CopyToFile(newFilename, privateFilePerms, reader, size)
	if err != nil {
		return err
	}
	defer os.Remove(newFilename)
	if err := os.Rename(filename, oldFilename); err != nil {
		return err
	}
	if err := os.Rename(newFilename, filename); err != nil {
		os.Rename(oldFilename, filename)
		return err
	}
	return nil
}

func (m *Manager) restoreVmFromSnapshot(ipAddr net.IP,
	authInfo *srpc.AuthInformation, forceIfNotStopped bool) error {
	vm, err := m.getVmLockAndAuth(ipAddr, authInfo, nil)
	if err != nil {
		return err
	}
	defer vm.mutex.Unlock()
	if vm.State != proto.StateStopped {
		if !forceIfNotStopped {
			return errors.New("VM is not stopped")
		}
	}
	for _, volume := range vm.VolumeLocations {
		snapshotFilename := volume.Filename + ".snapshot"
		if err := os.Rename(snapshotFilename, volume.Filename); err != nil {
			if !os.IsNotExist(err) {
				return err
			}
		}
	}
	return nil
}

func (m *Manager) restoreVmImage(ipAddr net.IP,
	authInfo *srpc.AuthInformation) error {
	vm, err := m.getVmLockAndAuth(ipAddr, authInfo, nil)
	if err != nil {
		return err
	}
	defer vm.mutex.Unlock()
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
	vm.writeAndSendInfo()
	return nil
}

func (m *Manager) restoreVmUserData(ipAddr net.IP,
	authInfo *srpc.AuthInformation) error {
	vm, err := m.getVmLockAndAuth(ipAddr, authInfo, nil)
	if err != nil {
		return err
	}
	defer vm.mutex.Unlock()
	filename := path.Join(vm.dirname, "user-data.raw")
	oldFilename := filename + ".old"
	return os.Rename(oldFilename, filename)
}

func (m *Manager) sendVmInfo(ipAddress string, vm *proto.VmInfo) {
	if ipAddress != "0.0.0.0" {
		if vm == nil { // GOB cannot encode a nil value in a map.
			vm = new(proto.VmInfo)
		}
		m.sendUpdateWithLock(proto.Update{
			HaveVMs: true,
			VMs:     map[string]*proto.VmInfo{ipAddress: vm},
		})
	}
}

func (m *Manager) snapshotVm(ipAddr net.IP, authInfo *srpc.AuthInformation,
	forceIfNotStopped, snapshotRootOnly bool) error {
	vm, err := m.getVmLockAndAuth(ipAddr, authInfo, nil)
	if err != nil {
		return err
	}
	defer vm.mutex.Unlock()
	// TODO(rgooch): First check for sufficient free space.
	if vm.State != proto.StateStopped {
		if !forceIfNotStopped {
			return errors.New("VM is not stopped")
		}
	}
	if err := vm.discardSnapshot(); err != nil {
		return err
	}
	doCleanup := true
	defer func() {
		if doCleanup {
			vm.discardSnapshot()
		}
	}()
	for index, volume := range vm.VolumeLocations {
		snapshotFilename := volume.Filename + ".snapshot"
		if index == 0 || !snapshotRootOnly {
			err := fsutil.CopyFile(snapshotFilename, volume.Filename,
				privateFilePerms)
			if err != nil {
				return err
			}
		}
	}
	doCleanup = false
	return nil
}

func (m *Manager) startVm(ipAddr net.IP, authInfo *srpc.AuthInformation,
	accessToken []byte, dhcpTimeout time.Duration) (bool, error) {
	vm, err := m.getVmLockAndAuth(ipAddr, authInfo, accessToken)
	if err != nil {
		return false, err
	}
	defer vm.mutex.Unlock()
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
		return vm.startManaging(dhcpTimeout, false)
	case proto.StateDestroying:
		return false, errors.New("VM is destroying")
	case proto.StateMigrating:
		return false, errors.New("VM is migrating")
	default:
		return false, errors.New("unknown state: " + vm.State.String())
	}
	return false, nil
}

func (m *Manager) stopVm(ipAddr net.IP, authInfo *srpc.AuthInformation,
	accessToken []byte) error {
	vm, err := m.getVmLockAndAuth(ipAddr, authInfo, accessToken)
	if err != nil {
		return err
	}
	doUnlock := true
	defer func() {
		if doUnlock {
			vm.mutex.Unlock()
		}
	}()
	switch vm.State {
	case proto.StateStarting:
		return errors.New("VM is starting")
	case proto.StateRunning:
		if len(vm.Address.IpAddress) < 1 {
			return errors.New("cannot stop VM with externally managed lease")
		}
		stoppedNotifier := make(chan struct{}, 1)
		vm.stoppedNotifier = stoppedNotifier
		vm.setState(proto.StateStopping)
		vm.commandChannel <- "system_powerdown"
		time.AfterFunc(time.Second*15, vm.kill)
		vm.mutex.Unlock()
		doUnlock = false
		<-stoppedNotifier
	case proto.StateStopping:
		return errors.New("VM is stopping")
	case proto.StateStopped, proto.StateFailedToStart:
		return errors.New("VM is already stopped")
	case proto.StateDestroying:
		return errors.New("VM is destroying")
	case proto.StateMigrating:
		return errors.New("VM is migrating")
	default:
		return errors.New("unknown state: " + vm.State.String())
	}
	return nil
}

func (m *Manager) unregisterVmMetadataNotifier(ipAddr net.IP,
	pathChannel chan<- string) error {
	vm, err := m.getVmAndLock(ipAddr)
	if err != nil {
		return err
	}
	defer vm.mutex.Unlock()
	delete(vm.metadataChannels, pathChannel)
	return nil
}

func (vm *vmInfoType) autoDestroy() {
	vm.logger.Println("VM was not acknowledged, destroying")
	authInfo := &srpc.AuthInformation{HaveMethodAccess: true}
	err := vm.manager.destroyVm(vm.Address.IpAddress, authInfo, nil)
	if err != nil {
		vm.logger.Println(err)
	}
}

func (vm *vmInfoType) changeIpAddress(ipAddress string) error {
	dirname := path.Join(vm.manager.StateDir, "VMs", ipAddress)
	if err := os.Rename(vm.dirname, dirname); err != nil {
		return err
	}
	vm.dirname = dirname
	for index, volume := range vm.VolumeLocations {
		parent := path.Dir(volume.DirectoryToCleanup)
		dirname := path.Join(parent, ipAddress)
		if err := os.Rename(volume.DirectoryToCleanup, dirname); err != nil {
			return err
		}
		vm.VolumeLocations[index] = volumeType{
			DirectoryToCleanup: dirname,
			Filename:           path.Join(dirname, path.Base(volume.Filename)),
		}
	}
	vm.logger.Printf("changing to new address: %s\n", ipAddress)
	vm.logger = prefixlogger.New(ipAddress+": ", vm.manager.Logger)
	vm.writeInfo()
	vm.manager.mutex.Lock()
	defer vm.manager.mutex.Unlock()
	delete(vm.manager.vms, vm.ipAddress)
	vm.ipAddress = ipAddress
	vm.manager.vms[vm.ipAddress] = vm
	vm.manager.sendUpdateWithLock(proto.Update{
		HaveVMs: true,
		VMs:     map[string]*proto.VmInfo{ipAddress: &vm.VmInfo},
	})
	return nil
}

func (vm *vmInfoType) checkAuth(authInfo *srpc.AuthInformation,
	accessToken []byte) error {
	if authInfo.HaveMethodAccess {
		return nil
	}
	if _, ok := vm.ownerUsers[authInfo.Username]; ok {
		return nil
	}
	if len(vm.accessToken) >= 32 && bytes.Equal(vm.accessToken, accessToken) {
		return nil
	}
	for _, ownerGroup := range vm.OwnerGroups {
		if _, ok := authInfo.GroupList[ownerGroup]; ok {
			return nil
		}
	}
	return errorNoAccessToResource
}

func (vm *vmInfoType) cleanup() {
	if vm == nil {
		return
	}
	select {
	case vm.commandChannel <- "quit":
	default:
	}
	m := vm.manager
	m.mutex.Lock()
	delete(m.vms, vm.ipAddress)
	if !vm.doNotWriteOrSend {
		m.sendVmInfo(vm.ipAddress, nil)
	}
	if !vm.Uncommitted {
		if err := m.releaseAddressInPoolWithLock(vm.Address); err != nil {
			m.Logger.Println(err)
		}
	}
	os.RemoveAll(vm.dirname)
	for _, volume := range vm.VolumeLocations {
		os.RemoveAll(volume.DirectoryToCleanup)
	}
	m.mutex.Unlock()
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
	vm.Volumes = []proto.Volume{{Size: size}}
	return setVolumeSize(vm.VolumeLocations[0].Filename, size)
}

func (vm *vmInfoType) delete() {
	select {
	case vm.accessTokenCleanupNotifier <- struct{}{}:
	default:
	}
	for ch := range vm.metadataChannels {
		close(ch)
	}
	vm.manager.DhcpServer.RemoveLease(vm.Address.IpAddress)
	vm.manager.mutex.Lock()
	delete(vm.manager.vms, vm.ipAddress)
	vm.manager.sendVmInfo(vm.ipAddress, nil)
	var err error
	if !vm.Uncommitted {
		err = vm.manager.releaseAddressInPoolWithLock(vm.Address)
	}
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

func (vm *vmInfoType) destroy() {
	select {
	case vm.commandChannel <- "quit":
	default:
	}
	vm.delete()
}

func (vm *vmInfoType) discardSnapshot() error {
	for _, volume := range vm.VolumeLocations {
		if err := os.Remove(volume.Filename + ".snapshot"); err != nil {
			if !os.IsNotExist(err) {
				return err
			}
		}
	}
	return nil
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
	stopTime := time.Now().Add(time.Minute * 5)
	for time.Until(stopTime) > 0 {
		select {
		case <-cancel:
			return
		default:
		}
		sleepUntil := time.Now().Add(time.Second)
		if vm.ipAddress == "0.0.0.0" {
			time.Sleep(time.Until(sleepUntil))
			continue
		}
		conn, err := net.DialTimeout("tcp", vm.ipAddress+":6910", time.Second*5)
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
		select {
		case vm.stoppedNotifier <- struct{}{}:
		default:
		}
	case proto.StateStopped:
		return
	case proto.StateDestroying:
		vm.delete()
		return
	case proto.StateMigrating:
		return
	default:
		vm.logger.Println("unknown state: " + vm.State.String())
	}
	close(vm.commandChannel)
}

func (vm *vmInfoType) setState(state proto.State) {
	vm.State = state
	if !vm.doNotWriteOrSend {
		vm.writeAndSendInfo()
	}
}

func (vm *vmInfoType) setupVolumes(rootSize uint64,
	request proto.CreateVmRequest) error {
	volumeDirectories, err := vm.manager.getVolumeDirectories(rootSize,
		request.SecondaryVolumes, request.SpreadVolumes)
	if err != nil {
		return err
	}
	volumeDirectory := path.Join(volumeDirectories[0], vm.ipAddress)
	os.RemoveAll(volumeDirectory)
	if err := os.MkdirAll(volumeDirectory, dirPerms); err != nil {
		return err
	}
	filename := path.Join(volumeDirectory, "root")
	vm.VolumeLocations = append(vm.VolumeLocations,
		volumeType{volumeDirectory, filename})
	for index := range request.SecondaryVolumes {
		volumeDirectory := path.Join(volumeDirectories[index+1], vm.ipAddress)
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

func (vm *vmInfoType) startManaging(dhcpTimeout time.Duration,
	haveManagerLock bool) (bool, error) {
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
	case proto.StateMigrating:
		return false, nil
	default:
		vm.logger.Println("unknown state: " + vm.State.String())
		return false, nil
	}
	vm.manager.DhcpServer.AddLease(vm.Address, vm.Hostname)
	monitorSock, err := net.Dial("unix", vm.monitorSockname)
	if err != nil {
		vm.logger.Debugf(0, "error connecting to: %s: %s\n",
			vm.monitorSockname, err)
		if err := vm.startVm(haveManagerLock); err != nil {
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
	if len(vm.Address.IpAddress) < 1 {
		// Must wait to see what IP address is given by external DHCP server.
		reqCh := vm.manager.DhcpServer.MakeRequestChannel(vm.Address.MacAddress)
		if dhcpTimeout < time.Minute {
			dhcpTimeout = time.Minute
		}
		timer := time.NewTimer(dhcpTimeout)
		select {
		case ipAddr := <-reqCh:
			timer.Stop()
			return false, vm.changeIpAddress(ipAddr.String())
		case <-timer.C:
			return true, errors.New("timed out on external lease")
		}
	}
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

func (vm *vmInfoType) startVm(haveManagerLock bool) error {
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
	if !haveManagerLock {
		vm.manager.mutex.RLock()
	}
	subnet, ok := vm.manager.subnets[vm.SubnetId]
	if !haveManagerLock {
		vm.manager.mutex.RUnlock()
	}
	if !ok {
		return fmt.Errorf("subnet: %s not found", vm.SubnetId)
	}
	var bridge string
	var vlanOption string
	if bridge, ok = vm.manager.VlanIdToBridge[subnet.VlanId]; !ok {
		if bridge, ok = vm.manager.VlanIdToBridge[0]; !ok {
			return fmt.Errorf("no usable bridge")
		} else {
			vlanOption = fmt.Sprintf(",vlan=%d", subnet.VlanId)
		}
	}
	tapFile, err := createTapDevice(bridge)
	if err != nil {
		return fmt.Errorf("error creating tap device: %s", err)
	}
	defer tapFile.Close()
	cmd := exec.Command("qemu-system-x86_64", "-machine", "pc,accel=kvm",
		"-nodefaults",
		"-name", vm.ipAddress,
		"-m", fmt.Sprintf("%dM", vm.MemoryInMiB),
		"-smp", fmt.Sprintf("cpus=%d", nCpus),
		"-net", "nic,model=virtio,macaddr="+vm.Address.MacAddress,
		"-net", "tap,fd=3"+vlanOption,
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
	for index, volume := range vm.VolumeLocations {
		var volumeFormat proto.VolumeFormat
		if index < len(vm.Volumes) {
			volumeFormat = vm.Volumes[index].Format
		}
		cmd.Args = append(cmd.Args,
			"-drive", "file="+volume.Filename+",format="+volumeFormat.String()+
				",if=virtio")
	}
	os.Remove(bootlogFilename)
	cmd.ExtraFiles = []*os.File{tapFile} // fd=3 for QEMU.
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error starting QEMU: %s: %s", err, output)
	}
	return nil
}

func (vm *vmInfoType) writeAndSendInfo() {
	if err := vm.writeInfo(); err != nil {
		vm.logger.Println(err)
		return
	}
	vm.manager.sendVmInfo(vm.ipAddress, &vm.VmInfo)
}

func (vm *vmInfoType) writeInfo() error {
	filename := path.Join(vm.dirname, "info.json")
	return json.WriteToFile(filename, publicFilePerms, "    ", vm)
}
