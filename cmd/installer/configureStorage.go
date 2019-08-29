// +build linux

package main

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	imageclient "github.com/Symantec/Dominator/imageserver/client"
	"github.com/Symantec/Dominator/lib/concurrent"
	"github.com/Symantec/Dominator/lib/cpusharer"
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/filesystem/util"
	"github.com/Symantec/Dominator/lib/format"
	"github.com/Symantec/Dominator/lib/fsutil"
	"github.com/Symantec/Dominator/lib/image"
	"github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/objectserver"
	objectclient "github.com/Symantec/Dominator/lib/objectserver/client"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/lib/wsyscall"
	fm_proto "github.com/Symantec/Dominator/proto/fleetmanager"
	installer_proto "github.com/Symantec/Dominator/proto/installer"
)

const (
	keyFile = "/etc/crypt.key"
)

type driveType struct {
	discarded bool
	devpath   string
	name      string
	size      uint64 // Bytes
}

func init() {
	gob.Register(&filesystem.RegularInode{})
	gob.Register(&filesystem.SymlinkInode{})
	gob.Register(&filesystem.SpecialInode{})
	gob.Register(&filesystem.DirectoryInode{})
}

func configureBootDrive(cpuSharer cpusharer.CpuSharer, drive *driveType,
	layout installer_proto.StorageLayout, bootPartition int, img *image.Image,
	objGetter objectserver.ObjectsGetter, logger log.DebugLogger) error {
	startTime := time.Now()
	if run("blkdiscard", *tmpRoot, logger, drive.devpath) == nil {
		drive.discarded = true
		logger.Printf("discarded %s in %s\n",
			drive.devpath, format.Duration(time.Since(startTime)))
	} else { // Erase old partition.
		if err := eraseStart(drive.devpath, logger); err != nil {
			return err
		}
	}
	args := []string{"-s", "-a", "cylinder", drive.devpath, "mklabel", "msdos"}
	unitSize := uint64(1 << 20)
	unitSuffix := "MiB"
	offsetInUnits := uint64(1)
	for _, partition := range layout.BootDriveLayout {
		sizeInUnits := partition.MinimumFreeBytes / unitSize
		if sizeInUnits*unitSize < partition.MinimumFreeBytes {
			sizeInUnits++
		}
		args = append(args, "mkpart", "primary", "ext2",
			strconv.FormatUint(offsetInUnits, 10)+unitSuffix,
			strconv.FormatUint(offsetInUnits+sizeInUnits, 10)+unitSuffix)
		offsetInUnits += sizeInUnits
	}
	args = append(args, "mkpart", "primary", "ext2",
		strconv.FormatUint(offsetInUnits, 10)+unitSuffix, "100%")
	args = append(args,
		"set", strconv.FormatInt(int64(bootPartition), 10), "boot", "on")
	if err := run("parted", *tmpRoot, logger, args...); err != nil {
		return err
	}
	// Prepare all file-systems concurrently, make them serially.
	concurrentState := concurrent.NewState(uint(
		len(layout.BootDriveLayout) + 1))
	var mkfsMutex sync.Mutex
	for index, partition := range layout.BootDriveLayout {
		device := partitionName(drive.devpath, index+1)
		partition := partition
		err := concurrentState.GoRun(func() error {
			return drive.makeFileSystem(cpuSharer, device, partition.MountPoint,
				"ext4", &mkfsMutex, false, logger)
		})
		if err != nil {
			return err
		}
	}
	concurrentState.GoRun(func() error {
		device := partitionName(drive.devpath, len(layout.BootDriveLayout)+1)
		return drive.makeFileSystem(cpuSharer, device,
			layout.ExtraMountPointsBasename+"0", "ext4", &mkfsMutex, true,
			logger)
	})
	if err := concurrentState.Reap(); err != nil {
		return err
	}
	// Mount all file-systems, except the data file-system.
	for index, partition := range layout.BootDriveLayout {
		device := partitionName(drive.devpath, index+1)
		err := mount(remapDevice(device, partition.MountPoint),
			filepath.Join(*mountPoint, partition.MountPoint), "ext4", logger)
		if err != nil {
			return err
		}
	}
	return installRoot(drive.devpath, img.FileSystem, objGetter, logger)
}

func configureDataDrive(cpuSharer cpusharer.CpuSharer, drive *driveType,
	index int, layout installer_proto.StorageLayout,
	logger log.DebugLogger) error {
	startTime := time.Now()
	if run("blkdiscard", *tmpRoot, logger, drive.devpath) == nil {
		drive.discarded = true
		logger.Printf("discarded %s in %s\n",
			drive.devpath, format.Duration(time.Since(startTime)))
	}
	dataMountPoint := layout.ExtraMountPointsBasename + strconv.FormatInt(
		int64(index), 10)
	return drive.makeFileSystem(cpuSharer, drive.devpath, dataMountPoint,
		"ext4", nil, true, logger)
}

func configureStorage(config fm_proto.GetMachineInfoResponse,
	logger log.DebugLogger) error {
	startTime := time.Now()
	var layout installer_proto.StorageLayout
	err := json.ReadFromFile(filepath.Join(*tftpDirectory,
		"storage-layout.json"),
		&layout)
	if err != nil {
		return err
	}
	var bootPartition, rootPartition int
	for index, partition := range layout.BootDriveLayout {
		switch partition.MountPoint {
		case "/":
			rootPartition = index + 1
		case "/boot":
			bootPartition = index + 1
		}
	}
	if rootPartition < 1 {
		return fmt.Errorf("no root partition specified in layout")
	}
	if bootPartition < 1 {
		bootPartition = rootPartition
	}
	drives, err := listDrives(logger)
	if err != nil {
		return err
	}
	rootDevice := partitionName(drives[0].devpath, rootPartition)
	randomKey, err := getRandomKey(16, logger)
	if err != nil {
		return err
	}
	img, client, err := getImage(logger)
	if err != nil {
		return err
	}
	defer client.Close()
	if img == nil {
		logger.Println("no image specified, skipping paritioning")
		return nil
	} else {
		if err := img.FileSystem.RebuildInodePointers(); err != nil {
			return err
		}
		imageSize := img.FileSystem.EstimateUsage(0)
		if layout.BootDriveLayout[rootPartition-1].MinimumFreeBytes <
			imageSize {
			layout.BootDriveLayout[rootPartition-1].MinimumFreeBytes = imageSize
		}
		layout.BootDriveLayout[rootPartition-1].MinimumFreeBytes += imageSize
	}
	objClient := objectclient.AttachObjectClient(client)
	defer objClient.Close()
	objGetter, err := createObjectsCache(img.FileSystem.GetObjects(), objClient,
		rootDevice, logger)
	if err != nil {
		return err
	}
	if err := installTmpRoot(img.FileSystem, objGetter, logger); err != nil {
		return err
	}
	err = run("modprobe", *tmpRoot, logger, "-a", "algif_skcipher", "dm_crypt")
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filepath.Join(*tmpRoot, keyFile), randomKey,
		fsutil.PrivateFilePerms)
	if err != nil {
		return err
	}
	for index := range randomKey { // Scrub key.
		randomKey[index] = 0
	}
	// Configure all drives concurrently, making file-systems.
	// Use concurrent package because of it's reaping cabability.
	// Use cpusharer package to limit CPU intensive operations.
	concurrentState := concurrent.NewState(uint(len(drives)))
	cpuSharer := cpusharer.NewFifoCpuSharer()
	err = concurrentState.GoRun(func() error {
		return configureBootDrive(cpuSharer, drives[0], layout, bootPartition,
			img, objGetter, logger)
	})
	if err != nil {
		return concurrentState.Reap()
	}
	for index, drive := range drives[1:] {
		drive := drive
		index := index + 1
		err := concurrentState.GoRun(func() error {
			return configureDataDrive(cpuSharer, drive, index, layout, logger)
		})
		if err != nil {
			break
		}
	}
	if err := concurrentState.Reap(); err != nil {
		return err
	}
	// Make table entries for the boot device file-systems, except data FS.
	fsTab := &bytes.Buffer{}
	cryptTab := &bytes.Buffer{}
	for index, partition := range layout.BootDriveLayout {
		device := partitionName(drives[0].devpath, index+1)
		err = drives[0].writeDeviceEntries(device, partition.MountPoint, "ext4",
			fsTab, cryptTab, uint(index+1))
		if err != nil {
			return err
		}
	}
	// Make table entries for data file-systems.
	for index, drive := range drives {
		checkCount := uint(2)
		var device string
		if index == 0 { // The boot device is partitioned.
			checkCount = uint(len(layout.BootDriveLayout) + 1)
			device = partitionName(drives[0].devpath,
				len(layout.BootDriveLayout)+1)
		} else { // Extra drives are used whole.
			device = drive.devpath
		}
		dataMountPoint := layout.ExtraMountPointsBasename + strconv.FormatInt(
			int64(index), 10)
		err = drive.writeDeviceEntries(device, dataMountPoint, "ext4", fsTab,
			cryptTab, checkCount)
		if err != nil {
			return err
		}
	}
	err = ioutil.WriteFile(filepath.Join(*mountPoint, "etc", "fstab"),
		fsTab.Bytes(), fsutil.PublicFilePerms)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filepath.Join(*mountPoint, "/etc", "crypttab"),
		cryptTab.Bytes(), fsutil.PublicFilePerms)
	if err != nil {
		return err
	}
	err = fsutil.CopyFile(filepath.Join(*mountPoint, keyFile),
		filepath.Join(*tmpRoot, keyFile), fsutil.PrivateFilePerms)
	if err != nil {
		return err
	}
	logdir := filepath.Join(*mountPoint, "var", "log", "installer")
	if err := os.MkdirAll(logdir, fsutil.DirPerms); err != nil {
		return err
	}
	if err := fsutil.CopyTree(logdir, *tftpDirectory); err != nil {
		return err
	}
	logger.Printf("configureStorage() took %s\n",
		format.Duration(time.Since(startTime)))
	return nil
}

func eraseStart(device string, logger log.DebugLogger) error {
	if *dryRun {
		logger.Debugf(0, "dry run: skipping erasure of: %s\n", device)
		return nil
	}
	logger.Debugf(0, "erasing start of: %s\n", device)
	file, err := os.OpenFile(device, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer file.Close()
	var buffer [65536]byte
	if _, err := file.Write(buffer[:]); err != nil {
		return err
	}
	return nil
}

func getImage(logger log.DebugLogger) (*image.Image, *srpc.Client, error) {
	data, err := ioutil.ReadFile(filepath.Join(*tftpDirectory, "imagename"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	imageName := strings.TrimSpace(string(data))
	data, err = ioutil.ReadFile(filepath.Join(*tftpDirectory, "imageserver"))
	if err != nil {
		return nil, nil, err
	}
	imageServerAddress := strings.TrimSpace(string(data))
	logger.Printf("dialing imageserver: %s\n", imageServerAddress)
	startTime := time.Now()
	client, err := srpc.DialHTTP("tcp", imageServerAddress, time.Second*15)
	if err != nil {
		return nil, nil, err
	}
	logger.Printf("dialed imageserver after: %s\n",
		format.Duration(time.Since(startTime)))
	startTime = time.Now()
	if img, _ := imageclient.GetImage(client, imageName); img != nil {
		logger.Debugf(0, "got image: %s in %s\n",
			imageName, format.Duration(time.Since(startTime)))
		return img, client, nil
	}
	streamName := imageName
	isDir, err := imageclient.CheckDirectory(client, streamName)
	if err != nil {
		client.Close()
		return nil, nil, err
	}
	if !isDir {
		streamName = filepath.Dir(streamName)
		isDir, err = imageclient.CheckDirectory(client, streamName)
		if err != nil {
			client.Close()
			return nil, nil, err
		}
	}
	if !isDir {
		client.Close()
		return nil, nil, fmt.Errorf("%s is not a directory", streamName)
	}
	imageName, err = imageclient.FindLatestImage(client, streamName, false)
	if err != nil {
		client.Close()
		return nil, nil, err
	}
	if imageName == "" {
		client.Close()
		return nil, nil, fmt.Errorf("no image found in: %s on: %s",
			streamName, imageServerAddress)
	}
	startTime = time.Now()
	if img, err := imageclient.GetImage(client, imageName); err != nil {
		client.Close()
		return nil, nil, err
	} else {
		logger.Debugf(0, "got image: %s in %s\n",
			imageName, format.Duration(time.Since(startTime)))
		return img, client, nil
	}
}

func getRandomKey(numBytes uint, logger log.DebugLogger) ([]byte, error) {
	logger.Printf("reading %d bytes from /dev/urandom\n", numBytes)
	startTime := time.Now()
	if file, err := os.Open("/dev/urandom"); err != nil {
		return nil, err
	} else {
		defer file.Close()
		buffer := make([]byte, numBytes)
		if nRead, err := file.Read(buffer); err != nil {
			return nil, err
		} else if nRead < len(buffer) {
			return nil, fmt.Errorf("read: %d random bytes", nRead)
		}
		logger.Printf("read %d bytes of random data after %s\n",
			numBytes, format.Duration(time.Since(startTime)))
		return buffer, nil
	}
}

func installRoot(device string, fileSystem *filesystem.FileSystem,
	objGetter objectserver.ObjectsGetter, logger log.DebugLogger) error {
	if *dryRun {
		logger.Debugln(0, "dry run: skipping installing root")
		return nil
	}
	logger.Debugln(0, "unpacking root")
	err := util.Unpack(fileSystem, objGetter, *mountPoint, logger)
	if err != nil {
		return err
	}
	err = wsyscall.Mount("/dev", filepath.Join(*mountPoint, "dev"), "",
		wsyscall.MS_BIND, "")
	if err != nil {
		return err
	}
	err = wsyscall.Mount("/proc", filepath.Join(*mountPoint, "proc"), "",
		wsyscall.MS_BIND, "")
	if err != nil {
		return err
	}
	err = wsyscall.Mount("/sys", filepath.Join(*mountPoint, "sys"), "",
		wsyscall.MS_BIND, "")
	if err != nil {
		return err
	}
	return util.MakeBootable(fileSystem, device, "rootfs", *mountPoint, "",
		true, logger)
}

func installTmpRoot(fileSystem *filesystem.FileSystem,
	objGetter objectserver.ObjectsGetter, logger log.DebugLogger) error {
	if fi, err := os.Stat(*tmpRoot); err == nil {
		if fi.IsDir() {
			logger.Debugln(0, "tmproot already exists, not installing")
			return nil
		}
	}
	if *dryRun {
		logger.Debugln(0, "dry run: skipping unpacking tmproot")
		return nil
	}
	logger.Debugln(0, "unpacking tmproot")
	if err := os.MkdirAll(*tmpRoot, fsutil.DirPerms); err != nil {
		return err
	}
	syscall.Unmount(filepath.Join(*tmpRoot, "sys"), 0)
	syscall.Unmount(filepath.Join(*tmpRoot, "proc"), 0)
	syscall.Unmount(filepath.Join(*tmpRoot, "dev"), 0)
	syscall.Unmount(*tmpRoot, 0)
	if err := wsyscall.Mount("none", *tmpRoot, "tmpfs", 0, ""); err != nil {
		return err
	}
	if err := util.Unpack(fileSystem, objGetter, *tmpRoot, logger); err != nil {
		return err
	}
	err := wsyscall.Mount("/dev", filepath.Join(*tmpRoot, "dev"), "",
		wsyscall.MS_BIND, "")
	if err != nil {
		return err
	}
	err = wsyscall.Mount("/proc", filepath.Join(*tmpRoot, "proc"), "",
		wsyscall.MS_BIND, "")
	if err != nil {
		return err
	}
	err = wsyscall.Mount("/sys", filepath.Join(*tmpRoot, "sys"), "",
		wsyscall.MS_BIND, "")
	if err != nil {
		return err
	}
	os.Symlink("/proc/mounts", filepath.Join(*tmpRoot, "etc", "mtab"))
	return nil
}

func listDrives(logger log.DebugLogger) ([]*driveType, error) {
	basedir := filepath.Join(*sysfsDirectory, "class", "block")
	file, err := os.Open(basedir)
	if err != nil {
		return nil, err
	}
	names, err := file.Readdirnames(-1)
	file.Close()
	if err != nil {
		return nil, err
	}
	sort.Strings(names)
	var drives []*driveType
	for _, name := range names {
		dirname := filepath.Join(basedir, name)
		if _, err := os.Stat(filepath.Join(dirname, "partition")); err == nil {
			logger.Debugf(2, "skipping partition: %s\n", name)
			continue
		}
		if _, err := os.Stat(filepath.Join(dirname, "device")); err != nil {
			if !os.IsNotExist(err) {
				return nil, err
			}
			logger.Debugf(2, "skipping non-device: %s\n", name)
			continue
		}
		if v, err := readInt(filepath.Join(dirname, "removable")); err != nil {
			return nil, err
		} else if v != 0 {
			logger.Debugf(2, "skipping removable device: %s\n", name)
			continue
		}
		if val, err := readInt(filepath.Join(dirname, "size")); err != nil {
			return nil, err
		} else {
			logger.Debugf(1, "found: %s %d GiB (%d GB)\n",
				name, val>>21, val<<9/1000000000)
			drives = append(drives, &driveType{
				devpath: filepath.Join("/dev", name),
				name:    name,
				size:    val << 9,
			})
		}
	}
	if len(drives) < 1 {
		return nil, fmt.Errorf("no drives found")
	}
	return drives, nil
}

func mount(source string, target string, fstype string,
	logger log.DebugLogger) error {
	if *dryRun {
		logger.Debugf(0, "dry run: skipping mount of %s on %s type=%s\n",
			source, target, fstype)
		return nil
	}
	logger.Debugf(0, "mount %s on %s type=%s\n", source, target, fstype)
	if err := os.MkdirAll(target, fsutil.DirPerms); err != nil {
		return err
	}
	return syscall.Mount(source, target, fstype, 0, "")
}

func partitionName(devpath string, partitionNumber int) string {
	leafName := filepath.Base(devpath)
	if strings.HasPrefix(leafName, "hd") ||
		strings.HasPrefix(leafName, "sd") {
		return devpath + strconv.FormatInt(int64(partitionNumber), 10)
	} else {
		return devpath + "p" + strconv.FormatInt(int64(partitionNumber), 10)
	}
}

func readInt(filename string) (uint64, error) {
	if file, err := os.Open(filename); err != nil {
		return 0, err
	} else {
		defer file.Close()
		var value uint64
		if nVal, err := fmt.Fscanf(file, "%d\n", &value); err != nil {
			return 0, err
		} else if nVal != 1 {
			return 0, fmt.Errorf("read %d values, expected 1", nVal)
		} else {
			return value, nil
		}
	}
}

func remapDevice(device, target string) string {
	if target == "/" {
		return device
	} else {
		return filepath.Join("/dev/mapper", filepath.Base(device))
	}
}

func unmountStorage(logger log.DebugLogger) error {
	syscall.Sync()
	time.Sleep(time.Millisecond * 100)
	file, err := os.Open("/proc/mounts")
	if err != nil {
		return err
	}
	defer file.Close()
	var mountPoints []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		} else {
			if strings.HasPrefix(fields[1], *mountPoint) {
				mountPoints = append(mountPoints, fields[1])
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	unmountedMainMountPoint := false
	for index := len(mountPoints) - 1; index >= 0; index-- {
		mntPoint := mountPoints[index]
		if err := syscall.Unmount(mntPoint, 0); err != nil {
			return fmt.Errorf("error unmounting: %s: %s", mntPoint, err)
		} else {
			logger.Debugf(2, "unmounted: %s\n", mntPoint)
		}
		if mntPoint == *mountPoint {
			unmountedMainMountPoint = true
		}
	}
	if !unmountedMainMountPoint {
		return errors.New("did not find main mount point to unmount")
	}
	syscall.Sync()
	return nil
}

func (drive driveType) cryptSetup(cpuSharer cpusharer.CpuSharer, device string,
	logger log.DebugLogger) error {
	cpuSharer.GrabCpu()
	defer cpuSharer.ReleaseCpu()
	startTime := time.Now()
	err := run("cryptsetup", *tmpRoot, logger, "--verbose",
		"--key-file", keyFile,
		"--cipher", "aes-xts-plain64", "--key-size", "512",
		"--hash", "sha512", "--iter-time", "5000", "--use-urandom",
		"luksFormat", device)
	if err != nil {
		return err
	}
	logger.Printf("formatted encrypted device %s in %s\n",
		device, time.Since(startTime))
	startTime = time.Now()
	if drive.discarded {
		err = run("cryptsetup", *tmpRoot, logger, "open", "--type", "luks",
			"--allow-discards",
			"--key-file", keyFile, device, filepath.Base(device))
	} else {
		err = run("cryptsetup", *tmpRoot, logger, "open", "--type", "luks",
			"--key-file", keyFile, device, filepath.Base(device))
	}
	if err != nil {
		return err
	}
	logger.Printf("opened encrypted device %s in %s\n",
		device, time.Since(startTime))
	return nil
}

func (drive driveType) makeFileSystem(cpuSharer cpusharer.CpuSharer,
	device, target, fstype string, mkfsMutex *sync.Mutex, data bool,
	logger log.DebugLogger) error {
	label := target
	erase := true
	if label == "/" {
		label = "rootfs"
		if drive.discarded {
			erase = false
		}
	} else {
		if err := drive.cryptSetup(cpuSharer, device, logger); err != nil {
			return err
		}
		device = filepath.Join("/dev/mapper", filepath.Base(device))
	}
	if erase {
		if err := eraseStart(device, logger); err != nil {
			return err
		}
	}
	var err error
	if mkfsMutex != nil {
		mkfsMutex.Lock()
	}
	startTime := time.Now()
	if data {
		err = run("mkfs.ext4", *tmpRoot, logger, "-i", "1048576", "-L", label,
			"-E", "lazy_itable_init=0,lazy_journal_init=0", device)
	} else {
		err = run("mkfs.ext4", *tmpRoot, logger, "-L", label,
			"-E", "lazy_itable_init=0,lazy_journal_init=0", device)
	}
	if mkfsMutex != nil {
		mkfsMutex.Unlock()
	}
	if err != nil {
		return err
	}
	logger.Printf("made file-system on %s in %s\n",
		device, time.Since(startTime))
	return nil
}

func (drive driveType) writeDeviceEntries(device, target, fstype string,
	fsTab, cryptTab io.Writer, checkOrder uint) error {
	label := target
	if label == "/" {
		label = "rootfs"
	} else {
		var options string
		if drive.discarded {
			options = "discard"
		}
		_, err := fmt.Fprintf(cryptTab, "%-15s %-23s %-15s %s\n",
			filepath.Base(device), device, keyFile, options)
		if err != nil {
			return err
		}
	}
	var fsFlags string
	if drive.discarded {
		fsFlags = "discard"
	}
	return util.WriteFstabEntry(fsTab, "LABEL="+label, target, fstype, fsFlags,
		0, checkOrder)
}
