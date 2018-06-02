package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/fsutil"
	"github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/lib/srpc/setupclient"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
)

const (
	dirPerms = syscall.S_IRWXU | syscall.S_IRGRP | syscall.S_IXGRP |
		syscall.S_IROTH | syscall.S_IXOTH
	privateFilePerms = syscall.S_IRUSR | syscall.S_IWUSR
)

func askForCommitDecision(client *srpc.Client, ipAddress net.IP) error {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Fprintf(os.Stderr, "Commit VM %s (commit/defer/abandon)? ",
			ipAddress)
		if response, err := reader.ReadString('\n'); err != nil {
			return fmt.Errorf("deferring, error reading input: %s", err)
		} else {
			switch response[:len(response)-1] {
			case "abandon":
				err := destroyVmOnHypervisorClient(client, ipAddress)
				if err != nil {
					return err
				}
				return fmt.Errorf("you abandoned your VM")
			case "commit":
				return commitVm(client, ipAddress)
			case "defer":
				return fmt.Errorf("you deferred committing your VM")
			}
		}
	}
}

func commitVm(client *srpc.Client, ipAddress net.IP) error {
	request := proto.CommitImportedVmRequest{ipAddress}
	var reply proto.CommitImportedVmResponse
	err := client.RequestReply("Hypervisor.CommitImportedVm", request, &reply)
	if err != nil {
		return err
	}
	return errors.New(reply.Error)
}

func importLocalVmSubcommand(args []string, logger log.DebugLogger) {
	if err := importLocalVm(args[0], args[1], logger); err != nil {
		fmt.Fprintf(os.Stderr, "Error importing VM: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func importLocalVm(infoFile, rootVolume string, logger log.DebugLogger) error {
	var vmInfo proto.VmInfo
	if err := json.ReadFromFile(infoFile, &vmInfo); err != nil {
		return err
	}
	return importLocalVmInfo(vmInfo, rootVolume, logger)
}

func importLocalVmInfo(vmInfo proto.VmInfo, rootVolume string,
	logger log.DebugLogger) error {
	importCookie, err := readImportCookie(logger)
	if err != nil {
		return err
	}
	hypervisor := fmt.Sprintf(":%d", *hypervisorPortNum)
	client, err := srpc.DialHTTP("tcp", hypervisor, 0)
	if err != nil {
		return err
	}
	defer client.Close()
	directories, err := listVolumeDirectories(client)
	if err != nil {
		return err
	}
	dirname := filepath.Join(directories[0], "import")
	if err := os.Mkdir(dirname, dirPerms); err != nil {
		if !os.IsExist(err) {
			return err
		}
	}
	dirname = filepath.Join(dirname, fmt.Sprintf("%d", os.Getpid()))
	if err := os.Mkdir(dirname, dirPerms); err != nil {
		return err
	}
	defer os.RemoveAll(dirname)
	logger.Debugf(0, "created: %s\n", dirname)
	rootFilename := filepath.Join(dirname, "root")
	if err := os.Link(rootVolume, rootFilename); err != nil {
		err = fsutil.CopyFile(rootFilename, rootVolume, privateFilePerms)
		if err != nil {
			return err
		}
	}
	request := proto.ImportLocalVmRequest{
		VerificationCookie: importCookie,
		VmInfo:             vmInfo,
		VolumeFilenames:    []string{rootFilename},
	}
	var reply proto.GetVmInfoResponse
	err = client.RequestReply("Hypervisor.ImportLocalVm", request, &reply)
	if err != nil {
		return err
	}
	if err := errors.New(reply.Error); err != nil {
		return err
	}
	logger.Debugln(0, "imported VM")
	os.RemoveAll(dirname)
	err = maybeWatchVm(client, hypervisor, vmInfo.Address.IpAddress, logger)
	if err != nil {
		return err
	}
	return askForCommitDecision(client, vmInfo.Address.IpAddress)
}

func listVolumeDirectories(client *srpc.Client) ([]string, error) {
	var request proto.ListVolumeDirectoriesRequest
	var reply proto.ListVolumeDirectoriesResponse
	err := client.RequestReply("Hypervisor.ListVolumeDirectories", request,
		&reply)
	if err != nil {
		return nil, err
	}
	if err := errors.New(reply.Error); err != nil {
		return nil, err
	}
	return reply.Directories, nil
}

func readImportCookie(logger log.DebugLogger) ([]byte, error) {
	importCookie, err := ioutil.ReadFile("/var/lib/hypervisor/import-cookie")
	if err != nil && os.IsPermission(err) {
		// Try again with sudo(8).
		args := make([]string, 0, len(os.Args)+1)
		if sudoPath, err := exec.LookPath("sudo"); err != nil {
			return nil, err
		} else {
			args = append(args, sudoPath)
		}
		if myPath, err := exec.LookPath(os.Args[0]); err != nil {
			return nil, err
		} else {
			args = append(args, myPath)
		}
		args = append(args, "-certDirectory", setupclient.GetCertDirectory())
		args = append(args, os.Args[1:]...)
		if err := syscall.Exec(args[0], args, os.Environ()); err != nil {
			return nil, errors.New("unable to Exec: " + err.Error())
		}
	}
	if err != nil {
		return nil, err
	}
	logger.Debugln(0, "have cookie")
	return importCookie, nil
}
