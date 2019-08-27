// +build linux

package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/Symantec/Dominator/lib/log"
)

type writeCloser struct{}

func create(filename string) (io.WriteCloser, error) {
	if *dryRun {
		return &writeCloser{}, nil
	}
	return os.Create(filename)
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

func run(name, chroot string, logger log.DebugLogger, args ...string) error {
	if *dryRun {
		logger.Debugf(0, "dry run: skipping: %s %s\n",
			name, strings.Join(args, " "))
		return nil
	}
	path, err := lookPath(chroot, name)
	if err != nil {
		return err
	}
	cmd := exec.Command(path, args...)
	if chroot != "" {
		cmd.Dir = "/"
		cmd.SysProcAttr = &syscall.SysProcAttr{Chroot: chroot}
		logger.Debugf(0, "running(chroot=%s): %s %s\n",
			chroot, name, strings.Join(args, " "))
	} else {
		logger.Debugf(0, "running: %s %s\n", name, strings.Join(args, " "))
	}
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error running: %s: %s", name, output)
	} else {
		return nil
	}
}

func (wc *writeCloser) Close() error {
	return nil
}

func (wc *writeCloser) Write(p []byte) (int, error) {
	return len(p), nil
}
