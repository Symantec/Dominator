package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"syscall"

	imageclient "github.com/Symantec/Dominator/imageserver/client"
	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/filesystem/util"
	"github.com/Symantec/Dominator/lib/fsutil"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/log/nulllogger"
	objectclient "github.com/Symantec/Dominator/lib/objectserver/client"
	"github.com/Symantec/Dominator/lib/srpc"
	fm_proto "github.com/Symantec/Dominator/proto/fleetmanager"
)

const (
	dirPerms = syscall.S_IRWXU | syscall.S_IRGRP | syscall.S_IXGRP |
		syscall.S_IROTH | syscall.S_IXOTH
)

func makeInstallerIsoSubcommand(args []string, logger log.DebugLogger) error {
	err := makeInstallerIso(args[0], args[1], logger)
	if err != nil {
		return fmt.Errorf("Error making installer ISO: %s", err)
	}
	return nil
}

func makeInstallerDirectory(hostname, rootDir string, logger log.DebugLogger) (
	*fm_proto.GetMachineInfoResponse, string, error) {
	fmCR := srpc.NewClientResource("tcp",
		fmt.Sprintf("%s:%d", *fleetManagerHostname, *fleetManagerPortNum))
	defer fmCR.ScheduleClose()
	imageClient, err := srpc.DialHTTP("tcp", fmt.Sprintf("%s:%d",
		*imageServerHostname, *imageServerPortNum), 0)
	if err != nil {
		return nil, "", err
	}
	defer imageClient.Close()
	info, _, configFiles, err := getInstallConfig(fmCR, imageClient, hostname,
		logger)
	if err != nil {
		return nil, "", err
	}
	err = unpackInstallerImage(rootDir, imageClient, nulllogger.New())
	if err != nil {
		return nil, "", err
	}
	initrdFile := filepath.Join(rootDir, "initrd.img")
	initrdRoot := filepath.Join(rootDir, "initrd.root")
	if err := unpackInitrd(initrdRoot, initrdFile); err != nil {
		return nil, "", err
	}
	configRoot := filepath.Join(initrdRoot, "tftpdata")
	if err := writeConfigFiles(configRoot, configFiles); err != nil {
		return nil, "", err
	}
	logger.Debugln(0, "building custom initrd with machine configuration")
	if err := packInitrd(initrdFile, initrdRoot); err != nil {
		return nil, "", err
	}
	return info, initrdFile, nil
}

func makeInstallerIso(hostname, dirname string, logger log.DebugLogger) error {
	rootDir, err := ioutil.TempDir("", "iso")
	if err != nil {
		return err
	}
	defer os.RemoveAll(rootDir)
	info, _, err := makeInstallerDirectory(hostname, rootDir, logger)
	if err != nil {
		return err
	}
	if info.Machine.IPMI.Hostname != "" {
		hostname = info.Machine.IPMI.Hostname
	}
	filename := filepath.Join(dirname, hostname+".iso")
	cmd := exec.Command("genisoimage", "-o", filename, "-b", "isolinux.bin",
		"-c", "boot.catalogue", "-no-emul-boot", "-boot-load-size", "4",
		"-boot-info-table", "-quiet", rootDir)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	if len(info.Machine.IPMI.HostIpAddress) > 0 {
		filename = filepath.Join(dirname,
			info.Machine.IPMI.HostIpAddress.String()+".iso")
		os.Remove(filename)
		if err := os.Symlink(hostname+".iso", filename); err != nil {
			return err
		}
	}
	fmt.Println(filename)
	return nil
}

func packInitrd(filename, rootDir string) error {
	paths, err := walkTree(rootDir)
	if err != nil {
		return err
	}
	sort.Strings(paths)
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := gzip.NewWriter(file)
	if err != nil {
		return err
	}
	defer writer.Close()
	// TODO(rgooch): Replace this with a library function using something like
	// github.com/cavaliercoder/go-cpio.
	cmd := exec.Command("cpio", "-o", "-H", "newc", "-R", "root.root",
		"--quiet")
	cmd.Dir = rootDir
	cmd.Stdout = writer
	cmd.Stderr = os.Stderr
	cmdStdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	for _, path := range paths {
		fmt.Fprintln(cmdStdin, path)
	}
	if err := cmdStdin.Close(); err != nil {
		return err
	}
	if err := cmd.Wait(); err != nil {
		return err
	}
	if err := os.RemoveAll(rootDir); err != nil {
		return err
	}
	return nil
}

func unpackInitrd(rootDir, filename string) error {
	if err := os.Mkdir(rootDir, dirPerms); err != nil {
		return err
	}
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	reader, err := gzip.NewReader(bufio.NewReader(file))
	if err != nil {
		return err
	}
	defer reader.Close()
	// TODO(rgooch): Replace this with a library function using something like
	// github.com/cavaliercoder/go-cpio.
	cmd := exec.Command("cpio", "-i", "--make-directories", "--numeric-uid-gid",
		"--preserve-modification-time", "--quiet")
	cmd.Dir = rootDir
	cmd.Stdin = reader
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	if err := os.Remove(filename); err != nil {
		return err
	}
	return nil
}

func unpackInstallerImage(rootDir string, imageClient *srpc.Client,
	logger log.DebugLogger) error {
	imageName, err := imageclient.FindLatestImage(imageClient,
		*installerImageStream, false)
	if err != nil {
		return err
	}
	if imageName == "" {
		return errors.New("no image found")
	}
	image, err := imageclient.GetImage(imageClient, imageName)
	if err != nil {
		return err
	}
	if euid := uint32(os.Geteuid()); euid != 0 {
		// Set the UID/GID to the user, otherwise unpacking will fail. This is a
		// bit dirty.
		// TODO(rgooch): Really want a util.UnpriviledgedUnpack() function.
		egid := uint32(os.Getegid())
		image.FileSystem.SetGid(egid)
		image.FileSystem.SetUid(euid)
		for _, inode := range image.FileSystem.InodeTable {
			inode.SetGid(egid)
			inode.SetUid(euid)
		}
	}
	image.FileSystem.RebuildInodePointers()
	objClient := objectclient.AttachObjectClient(imageClient)
	defer objClient.Close()
	err = util.Unpack(image.FileSystem, objClient, rootDir, logger)
	if err != nil {
		return err
	}
	return nil
}

func walkTree(rootDir string) ([]string, error) {
	rootLength := len(rootDir)
	var paths []string
	err := filepath.Walk(rootDir,
		func(path string, info os.FileInfo, err error) error {
			paths = append(paths, "."+path[rootLength:])
			return nil
		})
	return paths, err
}

func writeConfigFiles(rootDir string, configFiles map[string][]byte) error {
	if err := os.MkdirAll(rootDir, dirPerms); err != nil {
		return err
	}
	for name, data := range configFiles {
		err := fsutil.CopyToFile(filepath.Join(rootDir, name), filePerms,
			bytes.NewReader(data), uint64(len(data)))
		if err != nil {
			return err
		}
	}
	return nil
}
