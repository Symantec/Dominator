package util

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
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
)

type bootInfoType struct {
	InitrdImageFile string
	KernelImageFile string
	grubTemplate    *template.Template
}

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
	var err error
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
	}
	cmd := exec.Command("dumpe2fs", "-h", device)
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
		return 0, err
	}
	defer syscall.Close(fd)
	var blk uint64
	if err := ioctl(fd, BLKGETSIZE, uintptr(unsafe.Pointer(&blk))); err != nil {
		return 0, err
	}
	return blk << 9, nil
}

func ioctl(fd int, request, argp uintptr) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), request,
		argp)
	if errno != 0 {
		return os.NewSyscallError("ioctl", errno)
	}
	return nil
}

func makeAndWriteRoot(fs *filesystem.FileSystem,
	objectsGetter objectserver.ObjectsGetter, bootDevice, rootDevice string,
	makeBootableFlag bool, logger log.Logger) error {
	unsupportedOptions, err := getUnsupportedOptions(fs, objectsGetter)
	if err != nil {
		return err
	}
	var bootInfo *bootInfoType
	if makeBootableFlag {
		var err error
		bootInfo, err = getBootInfo(fs, objectsGetter)
		if err != nil {
			return err
		}
	}
	if err := makeRootFs(rootDevice, unsupportedOptions, logger); err != nil {
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
	os.RemoveAll(path.Join(mountPoint, "lost+found"))
	if err := Unpack(fs, objectsGetter, mountPoint, logger); err != nil {
		return err
	}
	if !makeBootableFlag {
		return nil
	}
	return bootInfo.makeBootable(bootDevice, mountPoint, logger)
}

func makeRootFs(deviceName string, unsupportedOptions []string,
	logger log.Logger) error {
	size, err := getDeviceSize(deviceName)
	if err != nil {
		return fmt.Errorf("error geting device size: %s: %s", deviceName, err)
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
	cmd := exec.Command("mkfs.ext4", "-L", "rootfs", "-i", "8192")
	if len(options) > 0 {
		cmd.Args = append(cmd.Args, "-O", strings.Join(options, ","))
	}
	cmd.Args = append(cmd.Args, deviceName, sizeString)
	logger.Printf("Making %s file-system\n", format.FormatBytes(size))
	startTime := time.Now()
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error making file-system on: %s: %s: %s",
			deviceName, err, output)
	}
	logger.Printf("Made file-system in %s\n",
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

func getBootInfo(fs *filesystem.FileSystem,
	objectsGetter objectserver.ObjectsGetter) (*bootInfoType, error) {
	bootDirectory, err := getBootDirectory(fs)
	if err != nil {
		return nil, err
	}
	bootInfo := &bootInfoType{}
	for _, dirent := range bootDirectory.EntryList {
		if strings.HasPrefix(dirent.Name, "initrd.img-") ||
			strings.HasPrefix(dirent.Name, "initramfs-") {
			if bootInfo.InitrdImageFile != "" {
				return nil, errors.New("multiple initrd images")
			}
			bootInfo.InitrdImageFile = "/boot/" + dirent.Name
		}
		if strings.HasPrefix(dirent.Name, "vmlinuz-") {
			if bootInfo.KernelImageFile != "" {
				return nil, errors.New("multiple kernel images")
			}
			bootInfo.KernelImageFile = "/boot/" + dirent.Name
		}
	}
	bootInfo.grubTemplate, err = template.New("grub").Parse(
		grubTemplateString)
	if err != nil {
		return nil, err
	}
	return bootInfo, nil
}

func (bootInfo *bootInfoType) makeBootable(deviceName string, rootDir string,
	logger log.Logger) error {
	startTime := time.Now()
	grubConfigFile := path.Join(rootDir, "boot", "grub", "grub.cfg")
	grubInstaller, err := exec.LookPath("grub-install")
	if err != nil {
		grubInstaller, err = exec.LookPath("grub2-install")
		if err != nil {
			return fmt.Errorf("cannot find GRUB installer: %s", err)
		}
		grubConfigFile = path.Join(rootDir, "boot", "grub2", "grub.cfg")
	}
	cmd := exec.Command(grubInstaller, "--boot-directory="+rootDir+"/boot",
		deviceName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error installing GRUB on: %s: %s: %s",
			deviceName, err, output)
	}
	logger.Printf("Installed GRUB in %s\n",
		format.Duration(time.Since(startTime)))
	file, err := os.Create(grubConfigFile)
	if err != nil {
		return fmt.Errorf("error creating GRUB config file: %s", err)
	}
	err = bootInfo.grubTemplate.Execute(file, bootInfo)
	if err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return ioutil.WriteFile(path.Join(rootDir, "etc", "fstab"),
		[]byte("LABEL=rootfs / ext4 defaults 0 1\n"),
		0644)
}

func writeToBlock(fs *filesystem.FileSystem,
	objectsGetter objectserver.ObjectsGetter, bootDevice string,
	tableType mbr.TableType, makeBootableFlag bool, logger log.Logger) error {
	if err := mbr.WriteDefault(bootDevice, tableType); err != nil {
		return err
	}
	if rootDevice, err := getRootPartition(bootDevice); err != nil {
		return err
	} else {
		return makeAndWriteRoot(fs, objectsGetter, bootDevice, rootDevice,
			makeBootableFlag, logger)
	}
}

func writeToFile(fs *filesystem.FileSystem,
	objectsGetter objectserver.ObjectsGetter, rawFilename string,
	perm os.FileMode, tableType mbr.TableType, minFreeSpace uint64,
	roundupPower uint64, makeBootableFlag, allocateBlocks bool,
	logger log.Logger) error {
	tmpFilename := rawFilename + "~"
	if file, err := os.OpenFile(tmpFilename, createFlags, perm); err != nil {
		return err
	} else {
		file.Close()
		defer os.Remove(tmpFilename)
	}
	usageEstimate := fs.EstimateUsage(0)
	minBytes := usageEstimate + usageEstimate>>3 // 12% extra for good luck.
	minBytes += minFreeSpace
	if roundupPower < 24 {
		roundupPower = 24 // 16 MiB.
	}
	imageUnits := minBytes >> roundupPower
	if imageUnits<<roundupPower < minBytes {
		imageUnits++
	}
	imageSize := imageUnits << roundupPower
	if err := os.Truncate(tmpFilename, int64(imageSize)); err != nil {
		return err
	}
	if allocateBlocks {
		if err := fsutil.Fallocate(tmpFilename, imageSize); err != nil {
			return err
		}
	}
	if err := mbr.WriteDefault(tmpFilename, tableType); err != nil {
		return err
	}
	cmd := exec.Command("losetup", "-fP", "--show", tmpFilename)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", err, output)
	}
	if output[len(output)-1] == '\n' {
		output = output[0 : len(output)-1]
	}
	loopDevice := string(output)
	defer exec.Command("losetup", "-d", loopDevice).Run()
	err = makeAndWriteRoot(fs, objectsGetter, loopDevice, loopDevice+"p1",
		makeBootableFlag, logger)
	if err != nil {
		return err
	}
	return os.Rename(tmpFilename, rawFilename)
}

func writeRaw(fs *filesystem.FileSystem,
	objectsGetter objectserver.ObjectsGetter, rawFilename string,
	perm os.FileMode, tableType mbr.TableType, minFreeSpace uint64,
	roundupPower uint64, makeBootableFlag, allocateBlocks bool,
	logger log.Logger) error {
	if isBlock, err := checkIsBlock(rawFilename); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	} else if isBlock {
		return writeToBlock(fs, objectsGetter, rawFilename, tableType,
			makeBootableFlag, logger)
	}
	return writeToFile(fs, objectsGetter, rawFilename, perm, tableType,
		minFreeSpace, roundupPower, makeBootableFlag, allocateBlocks, logger)
}

const grubTemplateString string = `# Generated from simple template.
set default="0"
if loadfont unicode ; then
  set gfxmode=auto
  insmod all_video
  insmod gfxterm
fi
terminal_output gfxterm
set timeout=0
set menu_color_normal=cyan/blue
set menu_color_highlight=white/blue

menuentry 'Linux' 'Solitary Linux' {
        insmod gzio
        insmod part_msdos
        insmod ext2
        echo    'Loading Linux {{.KernelImageFile}} ...'
        linux   {{.KernelImageFile}} root=LABEL=rootfs ro console=ttyS0,115200n8 console=tty0 net.ifnames=0
        echo    'Loading initial ramdisk ...'
        initrd  {{.InitrdImageFile}}
}
`
