package util

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"text/template"
	"time"
	"unsafe"

	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/format"
	"github.com/Symantec/Dominator/lib/fsutil"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/mbr"
	"github.com/Symantec/Dominator/lib/objectserver"
	"github.com/Symantec/Dominator/lib/wsyscall"
)

const (
	BLKGETSIZE  = 0x00001260
	createFlags = os.O_CREATE | os.O_TRUNC | os.O_RDWR
)

var (
	mutex               sync.Mutex
	defaultMkfsFeatures map[string]struct{} // Key: feature name.
	grubTemplate        = template.Must(template.New("grub").Parse(
		grubTemplateString))
)

func checkIfPartition(device string) (bool, error) {
	if isBlock, err := checkIsBlock(device); err != nil {
		if !os.IsNotExist(err) {
			return false, err
		}
		return false, nil
	} else if !isBlock {
		return false, fmt.Errorf("%s is not a block device", device)
	} else {
		return true, nil
	}
}

func checkIsBlock(filename string) (bool, error) {
	if fi, err := os.Stat(filename); err != nil {
		if !os.IsNotExist(err) {
			return false, fmt.Errorf("error stating: %s: %s", filename, err)
		}
		return false, err
	} else {
		return fi.Mode()&os.ModeDevice == os.ModeDevice, nil
	}
}

func findExecutable(rootDir, file string) error {
	if d, err := os.Stat(filepath.Join(rootDir, file)); err != nil {
		return err
	} else {
		if m := d.Mode(); !m.IsDir() && m&0111 != 0 {
			return nil
		}
		return os.ErrPermission
	}
}

func getBootDirectory(fs *filesystem.FileSystem) (
	*filesystem.DirectoryInode, error) {
	if fs.EntriesByName == nil {
		fs.BuildEntryMap()
	}
	dirent, ok := fs.EntriesByName["boot"]
	if !ok {
		return nil, errors.New("missing /boot directory")
	}
	bootDirectory, ok := dirent.Inode().(*filesystem.DirectoryInode)
	if !ok {
		return nil, errors.New("/boot is not a directory")
	}
	return bootDirectory, nil
}

func getDefaultMkfsFeatures(device, size string, logger log.Logger) (
	map[string]struct{}, error) {
	mutex.Lock()
	defer mutex.Unlock()
	if defaultMkfsFeatures == nil {
		startTime := time.Now()
		logger.Println("Making calibration file-system")
		cmd := exec.Command("mkfs.ext4", "-L", "calibration-fs", "-i", "65536",
			device, size)
		if output, err := cmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf(
				"error making calibration file-system on: %s: %s: %s",
				device, err, output)
		}
		logger.Printf("Made calibration file-system in %s\n",
			format.Duration(time.Since(startTime)))
		cmd = exec.Command("dumpe2fs", "-h", device)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("error dumping file-system info: %s: %s",
				err, output)
		}
		defaultMkfsFeatures = make(map[string]struct{})
		for _, line := range strings.Split(string(output), "\n") {
			fields := strings.Fields(line)
			if len(fields) < 3 {
				continue
			}
			if fields[0] != "Filesystem" || fields[1] != "features:" {
				continue
			}
			for _, field := range fields[2:] {
				defaultMkfsFeatures[field] = struct{}{}
			}
			break
		}
		// Scrub out the calibration file-system.
		buffer := make([]byte, 65536)
		if file, err := os.OpenFile(device, os.O_WRONLY, 0); err == nil {
			file.Write(buffer)
			file.Close()
		}
	}
	return defaultMkfsFeatures, nil
}

func getUnsupportedOptions(fs *filesystem.FileSystem,
	objectsGetter objectserver.ObjectsGetter) ([]string, error) {
	bootDirectory, err := getBootDirectory(fs)
	if err != nil {
		return nil, err
	}
	dirent, ok := bootDirectory.EntriesByName["ext4.unsupported-features"]
	var unsupportedOptions []string
	if ok {
		if inode, ok := dirent.Inode().(*filesystem.RegularInode); ok {
			hashes := []hash.Hash{inode.Hash}
			objectsReader, err := objectsGetter.GetObjects(hashes)
			if err != nil {
				return nil, err
			}
			defer objectsReader.Close()
			size, reader, err := objectsReader.NextObject()
			if err != nil {
				return nil, err
			}
			defer reader.Close()
			if size > 1024 {
				return nil,
					errors.New("/boot/ext4.unsupported-features is too large")
			}
			for {
				var option string
				_, err := fmt.Fscanf(reader, "%s\n", &option)
				if err != nil {
					if err == io.EOF {
						break
					}
					return nil, err
				} else {
					unsupportedOptions = append(unsupportedOptions,
						strings.Map(sanitiseInput, option))
				}
			}
		}
	}
	return unsupportedOptions, nil
}

func getRootPartition(bootDevice string) (string, error) {
	if isPartition, err := checkIfPartition(bootDevice + "p1"); err != nil {
		return "", err
	} else if isPartition {
		return bootDevice + "p1", nil
	}
	if isPartition, err := checkIfPartition(bootDevice + "1"); err != nil {
		return "", err
	} else if !isPartition {
		return "", errors.New("no root partition found")
	} else {
		return bootDevice + "1", nil
	}
}

func getDeviceSize(device string) (uint64, error) {
	fd, err := syscall.Open(device, os.O_RDONLY|syscall.O_CLOEXEC, 0666)
	if err != nil {
		return 0, fmt.Errorf("error opening: %s: %s", device, err)
	}
	defer syscall.Close(fd)
	var statbuf syscall.Stat_t
	if err := syscall.Fstat(fd, &statbuf); err != nil {
		return 0, fmt.Errorf("error stating: %s: %s\n", device, err)
	} else if statbuf.Mode&syscall.S_IFMT != syscall.S_IFBLK {
		return 0, fmt.Errorf("%s is not a block device, mode: %0o",
			device, statbuf.Mode)
	}
	var blk uint64
	err = wsyscall.Ioctl(fd, BLKGETSIZE, uintptr(unsafe.Pointer(&blk)))
	if err != nil {
		return 0, fmt.Errorf("error geting device size: %s: %s", device, err)
	}
	return blk << 9, nil
}

func lookPath(rootDir, file string) (string, error) {
	if strings.Contains(file, "/") {
		if err := findExecutable(rootDir, file); err != nil {
			return "", err
		}
		return file, nil
	}
	path := os.Getenv("PATH")
	for _, dir := range filepath.SplitList(path) {
		if dir == "" {
			dir = "." // Unix shell semantics: path element "" means "."
		}
		path := filepath.Join(dir, file)
		if err := findExecutable(rootDir, path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("(chroot=%s) %s not found in PATH", rootDir, file)
}

func makeAndWriteRoot(fs *filesystem.FileSystem,
	objectsGetter objectserver.ObjectsGetter, bootDevice, rootDevice string,
	options WriteRawOptions, logger log.DebugLogger) error {
	unsupportedOptions, err := getUnsupportedOptions(fs, objectsGetter)
	if err != nil {
		return err
	}
	var bootInfo *BootInfoType
	if options.RootLabel == "" {
		options.RootLabel = fmt.Sprintf("rootfs@%x", time.Now().Unix())
	}
	if options.InstallBootloader {
		var err error
		bootInfo, err = getBootInfo(fs, options.RootLabel, "net.ifnames=0")
		if err != nil {
			return err
		}
	}
	err = makeExt4fs(rootDevice, options.RootLabel, unsupportedOptions, 8192,
		logger)
	if err != nil {
		return err
	}
	mountPoint, err := ioutil.TempDir("", "write-raw-image")
	if err != nil {
		return err
	}
	defer os.RemoveAll(mountPoint)
	err = wsyscall.Mount(rootDevice, mountPoint, "ext4", 0, "")
	if err != nil {
		return fmt.Errorf("error mounting: %s", rootDevice)
	}
	defer syscall.Unmount(mountPoint, 0)
	os.RemoveAll(filepath.Join(mountPoint, "lost+found"))
	if err := Unpack(fs, objectsGetter, mountPoint, logger); err != nil {
		return err
	}
	if options.WriteFstab {
		err := writeRootFstabEntry(mountPoint, options.RootLabel)
		if err != nil {
			return err
		}
	}
	if options.InstallBootloader {
		err := bootInfo.installBootloader(bootDevice, mountPoint,
			options.RootLabel, options.DoChroot, logger)
		if err != nil {
			return err
		}
	}
	return nil
}

func makeBootable(fs *filesystem.FileSystem,
	deviceName, rootLabel, rootDir, kernelOptions string,
	doChroot bool, logger log.DebugLogger) error {
	if err := writeRootFstabEntry(rootDir, rootLabel); err != nil {
		return err
	}
	if bootInfo, err := getBootInfo(fs, rootLabel, kernelOptions); err != nil {
		return err
	} else {
		return bootInfo.installBootloader(deviceName, rootDir, rootLabel,
			doChroot, logger)
	}
}

func makeExt4fs(deviceName, label string, unsupportedOptions []string,
	bytesPerInode uint64, logger log.Logger) error {
	size, err := getDeviceSize(deviceName)
	if err != nil {
		return err
	}
	sizeString := strconv.FormatUint(size>>10, 10)
	var options []string
	if len(unsupportedOptions) > 0 {
		defaultFeatures, err := getDefaultMkfsFeatures(deviceName, sizeString,
			logger)
		if err != nil {
			return err
		}
		for _, option := range unsupportedOptions {
			if _, ok := defaultFeatures[option]; ok {
				options = append(options, "^"+option)
			}
		}
	}
	cmd := exec.Command("mkfs.ext4")
	if bytesPerInode != 0 {
		cmd.Args = append(cmd.Args, "-i", strconv.FormatUint(bytesPerInode, 10))
	}
	if label != "" {
		cmd.Args = append(cmd.Args, "-L", label)
	}
	if len(options) > 0 {
		cmd.Args = append(cmd.Args, "-O", strings.Join(options, ","))
	}
	cmd.Args = append(cmd.Args, deviceName, sizeString)
	startTime := time.Now()
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error making file-system on: %s: %s: %s",
			deviceName, err, output)
	}
	logger.Printf("Made %s file-system on: %s in %s\n",
		format.FormatBytes(size), deviceName,
		format.Duration(time.Since(startTime)))
	return nil
}

func sanitiseInput(ch rune) rune {
	if 'a' <= ch && ch <= 'z' {
		return ch
	} else if '0' <= ch && ch <= '9' {
		return ch
	} else if ch == '_' {
		return ch
	} else {
		return -1
	}
}

func getBootInfo(fs *filesystem.FileSystem, rootLabel string,
	extraKernelOptions string) (*BootInfoType, error) {
	bootDirectory, err := getBootDirectory(fs)
	if err != nil {
		return nil, err
	}
	bootInfo := &BootInfoType{
		BootDirectory: bootDirectory,
		KernelOptions: MakeKernelOptions("LABEL="+rootLabel,
			extraKernelOptions),
	}
	for _, dirent := range bootDirectory.EntryList {
		if strings.HasPrefix(dirent.Name, "initrd.img-") ||
			strings.HasPrefix(dirent.Name, "initramfs-") {
			if bootInfo.InitrdImageFile != "" {
				return nil, errors.New("multiple initrd images")
			}
			bootInfo.InitrdImageDirent = dirent
			bootInfo.InitrdImageFile = "/boot/" + dirent.Name
		}
		if strings.HasPrefix(dirent.Name, "vmlinuz-") {
			if bootInfo.KernelImageFile != "" {
				return nil, errors.New("multiple kernel images")
			}
			bootInfo.KernelImageDirent = dirent
			bootInfo.KernelImageFile = "/boot/" + dirent.Name
		}
	}
	return bootInfo, nil
}

func (bootInfo *BootInfoType) installBootloader(deviceName string,
	rootDir, rootLabel string, doChroot bool, logger log.DebugLogger) error {
	startTime := time.Now()
	var bootDir, chrootDir string
	if doChroot {
		bootDir = "/boot"
		chrootDir = rootDir
	} else {
		bootDir = filepath.Join(rootDir, "boot")
	}
	grubConfigFile := filepath.Join(rootDir, "boot", "grub", "grub.cfg")
	grubInstaller, err := lookPath(chrootDir, "grub-install")
	if err != nil {
		grubInstaller, err = lookPath(chrootDir, "grub2-install")
		if err != nil {
			return fmt.Errorf("cannot find GRUB installer: %s", err)
		}
		grubConfigFile = filepath.Join(rootDir, "boot", "grub2", "grub.cfg")
	}
	cmd := exec.Command(grubInstaller, "--boot-directory="+bootDir, deviceName)
	if doChroot {
		cmd.Dir = "/"
		cmd.SysProcAttr = &syscall.SysProcAttr{Chroot: chrootDir}
		logger.Debugf(0, "running(chroot=%s): %s %s\n",
			chrootDir, cmd.Path, strings.Join(cmd.Args[1:], " "))
	} else {
		logger.Debugf(0, "running: %s %s\n",
			cmd.Path, strings.Join(cmd.Args[1:], " "))
	}
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error installing GRUB on: %s: %s: %s",
			deviceName, err, output)
	}
	logger.Printf("installed GRUB in %s\n",
		format.Duration(time.Since(startTime)))
	if err := bootInfo.writeGrubConfig(grubConfigFile); err != nil {
		return err
	}
	return bootInfo.writeGrubTemplate(grubConfigFile + ".template")
}

func (bootInfo *BootInfoType) writeGrubConfig(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("error creating GRUB config file: %s", err)
	}
	defer file.Close()
	if err := grubTemplate.Execute(file, bootInfo); err != nil {
		return err
	}
	return file.Close()
}

func (bootInfo *BootInfoType) writeGrubTemplate(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("error creating GRUB config file template: %s", err)
	}
	defer file.Close()
	if _, err := file.Write([]byte(grubTemplateString)); err != nil {
		return err
	}
	return file.Close()
}

func writeFstabEntry(writer io.Writer,
	source, mountPoint, fileSystemType, flags string,
	dumpFrequency, checkOrder uint) error {
	if flags == "" {
		flags = "defaults"
	}
	_, err := fmt.Fprintf(writer, "%-22s %-10s %-5s %-10s %d %d\n",
		source, mountPoint, fileSystemType, flags, dumpFrequency, checkOrder)
	return err
}

func writeToBlock(fs *filesystem.FileSystem,
	objectsGetter objectserver.ObjectsGetter, bootDevice string,
	tableType mbr.TableType, options WriteRawOptions,
	logger log.DebugLogger) error {
	if err := mbr.WriteDefault(bootDevice, tableType); err != nil {
		return err
	}
	if rootDevice, err := getRootPartition(bootDevice); err != nil {
		return err
	} else {
		return makeAndWriteRoot(fs, objectsGetter, bootDevice, rootDevice,
			options, logger)
	}
}

func writeToFile(fs *filesystem.FileSystem,
	objectsGetter objectserver.ObjectsGetter, rawFilename string,
	perm os.FileMode, tableType mbr.TableType, options WriteRawOptions,
	logger log.DebugLogger) error {
	tmpFilename := rawFilename + "~"
	if file, err := os.OpenFile(tmpFilename, createFlags, perm); err != nil {
		return err
	} else {
		file.Close()
		defer os.Remove(tmpFilename)
	}
	usageEstimate := fs.EstimateUsage(0)
	minBytes := usageEstimate + usageEstimate>>3 // 12% extra for good luck.
	minBytes += options.MinimumFreeBytes
	if options.RoundupPower < 24 {
		options.RoundupPower = 24 // 16 MiB.
	}
	imageUnits := minBytes >> options.RoundupPower
	if imageUnits<<options.RoundupPower < minBytes {
		imageUnits++
	}
	imageSize := imageUnits << options.RoundupPower
	if err := os.Truncate(tmpFilename, int64(imageSize)); err != nil {
		return err
	}
	if err := mbr.WriteDefault(tmpFilename, tableType); err != nil {
		return err
	}
	loopDevice, err := fsutil.LoopbackSetup(tmpFilename)
	if err != nil {
		return err
	}
	defer fsutil.LoopbackDelete(loopDevice)
	err = makeAndWriteRoot(fs, objectsGetter, loopDevice, loopDevice+"p1",
		options, logger)
	if options.AllocateBlocks { // mkfs discards blocks, so do this after.
		if err := fsutil.Fallocate(tmpFilename, imageSize); err != nil {
			return err
		}
	}
	if err != nil {
		return err
	}
	return os.Rename(tmpFilename, rawFilename)
}

func writeRaw(fs *filesystem.FileSystem,
	objectsGetter objectserver.ObjectsGetter, rawFilename string,
	perm os.FileMode, tableType mbr.TableType, options WriteRawOptions,
	logger log.DebugLogger) error {
	if isBlock, err := checkIsBlock(rawFilename); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	} else if isBlock {
		return writeToBlock(fs, objectsGetter, rawFilename, tableType,
			options, logger)
	}
	return writeToFile(fs, objectsGetter, rawFilename, perm, tableType,
		options, logger)
}

func writeRootFstabEntry(rootDir, rootLabel string) error {
	file, err := os.Create(filepath.Join(rootDir, "etc", "fstab"))
	if err != nil {
		return err
	} else {
		defer file.Close()
		return writeFstabEntry(file, "LABEL="+rootLabel, "/", "ext4",
			"", 0, 1)
	}
}

const grubTemplateString string = `# Generated from simple template.
insmod serial
serial --unit=0 --speed=115200
terminal_output serial
set timeout=0

menuentry 'Linux' 'Solitary Linux' {
        insmod gzio
        insmod part_msdos
        insmod ext2
        echo    'Loading Linux {{.KernelImageFile}} ...'
        linux   {{.KernelImageFile}} {{.KernelOptions}}
        echo    'Loading initial ramdisk ...'
        initrd  {{.InitrdImageFile}}
}
`
