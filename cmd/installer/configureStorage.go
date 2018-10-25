package main

import (
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"

	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/filesystem/util"
	"github.com/Symantec/Dominator/lib/image"
	"github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/objectserver"
	objectclient "github.com/Symantec/Dominator/lib/objectserver/client"
	_ "github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/lib/wsyscall"
	installer_proto "github.com/Symantec/Dominator/proto/installer"
)

const (
	dirPerms = syscall.S_IRWXU | syscall.S_IRGRP | syscall.S_IXGRP |
		syscall.S_IROTH | syscall.S_IXOTH
	filePerms = syscall.S_IRUSR | syscall.S_IWUSR | syscall.S_IRGRP |
		syscall.S_IROTH
)

type driveType struct {
	devpath string
	name    string
	size    uint64 // Bytes
}

func init() {
	gob.Register(&filesystem.RegularInode{})
	gob.Register(&filesystem.SymlinkInode{})
	gob.Register(&filesystem.SpecialInode{})
	gob.Register(&filesystem.DirectoryInode{})
}

func configureStorage(logger log.DebugLogger) error {
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
	rootDevice := drives[0].devpath +
		strconv.FormatInt(int64(rootPartition), 10)
	img, err := getImage(logger)
	if err != nil {
		return err
	}
	if img == nil {
		logger.Println("no image, skipping paritioning")
		return nil
	}
	objClient, err := getObjectClient(logger)
	if err != nil {
		return err
	}
	defer objClient.Close()
	objGetter, err := createObjectsCache(img.FileSystem.GetObjects(), objClient,
		rootDevice, logger)
	if err != nil {
		return err
	}
	if err := installTmpRoot(img.FileSystem, objGetter, logger); err != nil {
		return err
	}
	// Attempt to discard blocks on SSDs.
	for _, drive := range drives {
		run("blkdiscard", *tmpRoot, logger, drive.devpath)
	}
	// Partition boot device.
	err = eraseStart(drives[0].devpath, logger)
	if err != nil {
		return err
	}
	args := []string{"-s", "-a", "cylinder", drives[0].devpath,
		"mklabel", "msdos"}
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
	// Make and mount file-systems.
	if err := makeAndMount(rootDevice, "/", "ext4", logger); err != nil {
		return err
	}
	for index, partition := range layout.BootDriveLayout {
		if partition.MountPoint == "/" {
			continue
		}
		err := makeAndMount(
			drives[0].devpath+strconv.FormatInt(int64(index+1), 10),
			partition.MountPoint, "ext4", logger)
		if err != nil {
			return err
		}
	}
	for index, drive := range drives {
		var device string
		if index == 0 {
			device = drives[0].devpath +
				strconv.FormatInt(int64(len(layout.BootDriveLayout)+1), 10)
		} else {
			device = drive.devpath
		}
		err := makeAndMount(device,
			layout.ExtraMountPointsBasename+strconv.FormatInt(int64(index), 10),
			"ext4", logger)
		if err != nil {
			return err
		}
	}
	err = installRoot(drives[0].devpath, img.FileSystem, objGetter, logger)
	if err != nil {
		return err
	}
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

func getImage(logger log.DebugLogger) (*image.Image, error) {
	file, err := os.Open(filepath.Join(*tftpDirectory, "image.gob"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()
	decoder := gob.NewDecoder(file)
	var img image.Image
	if err := decoder.Decode(&img); err != nil {
		return nil, err
	}
	if err := img.FileSystem.RebuildInodePointers(); err != nil {
		return nil, err
	}
	return &img, nil
}

func getObjectClient(logger log.DebugLogger) (
	*objectclient.ObjectClient, error) {
	data, err := ioutil.ReadFile(
		filepath.Join(*tftpDirectory, "objectserver"))
	if err != nil {
		return nil, err
	}
	return objectclient.NewObjectClient(strings.TrimSpace(string(data))), nil
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
	err = util.MakeBootable(fileSystem, device, *mountPoint, true, logger)
	return nil
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
	if err := os.MkdirAll(*tmpRoot, dirPerms); err != nil {
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

func listDrives(logger log.DebugLogger) ([]driveType, error) {
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
	var drives []driveType
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
			drives = append(drives, driveType{
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

func makeAndMount(device, target, fstype string, logger log.DebugLogger) error {
	label := target
	if label == "/" {
		label = "rootfs"
	}
	if err := makeFileSystem(device, label, logger); err != nil {
		return err
	}
	return mount(device, filepath.Join(*mountPoint, target), fstype, logger)
}

func makeFileSystem(device, label string, logger log.DebugLogger) error {
	if err := eraseStart(device, logger); err != nil {
		return err
	}
	return run("mkfs.ext4", *tmpRoot, logger, "-L", label, device)
}

func mount(source string, target string, fstype string,
	logger log.DebugLogger) error {
	if *dryRun {
		logger.Debugf(0, "dry run: skipping mount of %s on %s type=%s\n",
			source, target, fstype)
		return nil
	}
	logger.Debugf(0, "mount %s on %s type=%s\n", source, target, fstype)
	if err := os.MkdirAll(target, dirPerms); err != nil {
		return err
	}
	return syscall.Mount(source, target, fstype, 0, "")
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
			return 0, fmt.Errorf("read %2 values, expected 1", nVal)
		} else {
			return value, nil
		}
	}
}
