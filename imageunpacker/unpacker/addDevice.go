package unpacker

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
)

var sysfsDirectory = "/sys/block"

func (u *Unpacker) addDevice(deviceId string) error {
	u.updateUsageTime()
	scannedDevices, err := scanDevices()
	if err != nil {
		return err
	}
	u.rwMutex.Lock()
	defer u.rwMutex.Unlock()
	defer u.updateUsageTimeWithLock()
	for device := range u.scannedDevices {
		delete(scannedDevices, device)
	}
	if len(scannedDevices) < 1 {
		return errors.New("no new devices found")
	}
	if len(scannedDevices) > 1 {
		return errors.New("too many new devices found")
	}
	var deviceName string
	for d := range scannedDevices {
		deviceName = d
	}
	device := deviceInfo{DeviceName: deviceName}
	if err := updateDeviceSize(&device); err != nil {
		return err
	}
	if err := partitionAndMkFs(deviceName); err != nil {
		return err
	}
	u.pState.Devices[deviceId] = device
	return u.writeStateWithLock()
}

func partitionAndMkFs(deviceName string) error {
	cmd := exec.Command("parted", "-s", "-a", "optimal", "/dev/"+deviceName,
		"mklabel", "msdos", "mkpart", "primary", "ext2", "0%", "100%")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error partitioning: %s: %s", err, output)
	}
	cmd = exec.Command("mkfs.ext4", "-L", "rootfs", "/dev/"+deviceName+"1")
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error making file-system: %s: %s", err, output)
	}
	return nil
}

func (u *Unpacker) prepareForAddDevice() error {
	scannedDevices, err := scanDevices()
	if err != nil {
		return err
	}
	u.rwMutex.Lock()
	defer u.rwMutex.Unlock()
	u.scannedDevices = scannedDevices
	return nil
}

func scanDevices() (map[string]struct{}, error) {
	file, err := os.Open(sysfsDirectory)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	names, err := file.Readdirnames(-1)
	if err != nil {
		return nil, err
	}
	scannedDevices := make(map[string]struct{})
	for _, name := range names {
		scannedDevices[name] = struct{}{}
	}
	return scannedDevices, nil
}

func updateDeviceSize(device *deviceInfo) error {
	deviceBlocks, err := readSysfsUint64(
		path.Join(sysfsDirectory, device.DeviceName, "size"))
	if err != nil {
		return err
	}
	device.size = deviceBlocks * 512
	return nil
}

func readSysfsUint64(filename string) (uint64, error) {
	file, err := os.Open(filename)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	var value uint64
	nScanned, err := fmt.Fscanf(file, "%d", &value)
	if err != nil {
		return 0, err
	}
	if nScanned < 1 {
		return 0, errors.New(fmt.Sprintf("only read %d values from: %s",
			nScanned, filename))
	}
	return value, nil
}
