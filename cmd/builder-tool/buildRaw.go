// +build linux

package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"syscall"

	"github.com/Cloud-Foundations/Dominator/imagebuilder/builder"
	"github.com/Cloud-Foundations/Dominator/lib/filesystem/scanner"
	"github.com/Cloud-Foundations/Dominator/lib/filesystem/util"
	"github.com/Cloud-Foundations/Dominator/lib/fsutil"
	"github.com/Cloud-Foundations/Dominator/lib/hash"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/mbr"
	"github.com/Cloud-Foundations/Dominator/lib/wsyscall"
)

const createFlags = os.O_CREATE | os.O_TRUNC | os.O_RDWR

type dummyHasher struct{}

func buildRawFromManifestSubcommand(args []string,
	logger log.DebugLogger) error {
	if err := buildRawFromManifest(args[0], args[1], logger); err != nil {
		return fmt.Errorf("Error building RAW image from manifest: %s", err)
	}
	return nil
}

func buildRawFromManifest(manifestDir, rawFilename string,
	logger log.DebugLogger) error {
	if rawSize < 1<<20 {
		return fmt.Errorf("rawSize: %d too small", rawSize)
	}
	err := syscall.Mount("none", "/", "", syscall.MS_REC|syscall.MS_PRIVATE, "")
	if err != nil {
		return fmt.Errorf("error making mounts private: %s", err)
	}
	srpcClient := getImageServerClient()
	buildLog := &bytes.Buffer{}
	tmpFilename := rawFilename + "~"
	file, err := os.OpenFile(tmpFilename, createFlags, fsutil.PrivateFilePerms)
	if err != nil {
		return err
	}
	file.Close()
	defer os.Remove(tmpFilename)
	if err := os.Truncate(tmpFilename, int64(rawSize)); err != nil {
		return err
	}
	if err := mbr.WriteDefault(tmpFilename, mbr.TABLE_TYPE_MSDOS); err != nil {
		return err
	}
	loopDevice, err := fsutil.LoopbackSetup(tmpFilename)
	if err != nil {
		return err
	}
	defer fsutil.LoopbackDelete(loopDevice)
	rootDevice := loopDevice + "p1"
	rootLabel := "root@test"
	err = util.MakeExt4fs(rootDevice, rootLabel, nil, 0, logger)
	if err != nil {
		return err
	}
	rootDir, err := ioutil.TempDir("", "rootfs")
	if err != nil {
		return err
	}
	defer os.Remove(rootDir)
	err = wsyscall.Mount(rootDevice, rootDir, "ext4", 0, "")
	if err != nil {
		return fmt.Errorf("error mounting: %s", rootDevice)
	}
	defer syscall.Unmount(rootDir, 0)
	err = builder.UnpackImageAndProcessManifest(srpcClient, manifestDir,
		rootDir, bindMounts, buildLog)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error processing manifest: %s\n", err)
		io.Copy(os.Stderr, buildLog)
		os.Exit(1)
	}
	fs, err := scanner.ScanFileSystem(rootDir, nil, nil, nil, &dummyHasher{},
		nil)
	err = util.MakeBootable(&fs.FileSystem, loopDevice, rootLabel, rootDir,
		"net.ifnames=0", false, logger)
	if err != nil {
		return err
	}
	err = fsutil.CopyToFile("build.log", filePerms, buildLog,
		uint64(buildLog.Len()))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing build log: %s\n", err)
		os.Exit(1)
	}
	return os.Rename(tmpFilename, rawFilename)
}

func (h *dummyHasher) Hash(reader io.Reader, length uint64) (hash.Hash, error) {
	return hash.Hash{}, nil
}
