package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	imageclient "github.com/Symantec/Dominator/imageserver/client"
	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/log/nulllogger"
	"github.com/Symantec/Dominator/lib/srpc"
	hyper_proto "github.com/Symantec/Dominator/proto/hypervisor"
)

func reinstallSubcommand(args []string, logger log.DebugLogger) {
	err := reinstall(logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reinstalling: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func reinstall(logger log.DebugLogger) error {
	kexec, err := exec.LookPath("kexec")
	if err != nil {
		return err
	}
	cmd := exec.Command("hostname", "-f")
	var hostname string
	if stdout, err := cmd.Output(); err != nil {
		return err
	} else {
		hostname = strings.TrimSpace(string(stdout))
	}
	fmCR := srpc.NewClientResource("tcp",
		fmt.Sprintf("%s:%d", *fleetManagerHostname, *fleetManagerPortNum))
	defer fmCR.ScheduleClose()
	info, err := getInfoForMachine(fmCR, hostname)
	if err != nil {
		return err
	}
	imageName := info.Machine.Tags["RequiredImage"]
	subnets := make([]*hyper_proto.Subnet, 0, len(info.Subnets))
	for _, subnet := range info.Subnets {
		if subnet.VlanId == 0 {
			subnets = append(subnets, subnet)
		}
	}
	if len(subnets) < 1 {
		return errors.New("no non-VLAN subnets known")
	}
	networkEntries := getNetworkEntries(info)
	hostAddresses := getHostAddress(networkEntries)
	if len(hostAddresses) < 1 {
		return errors.New("no IP and MAC addresses known for host")
	}
	imageClient, err := srpc.DialHTTP("tcp", fmt.Sprintf("%s:%d",
		*imageServerHostname, *imageServerPortNum), 0)
	if err != nil {
		return err
	}
	defer imageClient.Close()
	if imageName != "" {
		img, err := imageclient.GetImage(imageClient, imageName)
		if err != nil {
			return err
		}
		if img == nil {
			return fmt.Errorf("image: %s does not exist", imageName)
		}
		if len(img.FileSystem.InodeTable) < 1000 {
			return fmt.Errorf("only %d inodes, this is likely not bootable",
				len(img.FileSystem.InodeTable))
		}
	}
	configFiles, err := makeConfigFiles(info, imageName, networkEntries)
	if err != nil {
		return err
	}
	rootDir, err := ioutil.TempDir("", "iso")
	if err != nil {
		return err
	}
	defer os.RemoveAll(rootDir)
	if err := unpackImage(rootDir, imageClient, nulllogger.New()); err != nil {
		return err
	}
	initrdFile := filepath.Join(rootDir, "initrd.img")
	initrdRoot := filepath.Join(rootDir, "initrd.root")
	if err := unpackInitrd(initrdRoot, initrdFile); err != nil {
		return err
	}
	configRoot := filepath.Join(initrdRoot, "tftpdata")
	if err := writeConfigFiles(configRoot, configFiles); err != nil {
		return err
	}
	if err := packInitrd(initrdFile, initrdRoot); err != nil {
		return err
	}
	logger.Println("running kexec in 5 seconds...")
	time.Sleep(time.Second * 5)
	var command string
	var args []string
	if os.Geteuid() == 0 {
		command = kexec
	} else {
		command = "sudo"
		args = []string{kexec}
	}
	args = append(args, "-l", filepath.Join(rootDir, "vmlinuz"),
		"--append=console=tty0 console=ttyS0,115200n8 net.ifnames=0",
		"--console-serial", "--serial-baud=115200",
		"--initrd="+initrdFile, "-f")
	cmd = exec.Command(command, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
