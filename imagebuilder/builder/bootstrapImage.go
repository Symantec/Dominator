package builder

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/Symantec/Dominator/lib/filter"
	"github.com/Symantec/Dominator/lib/format"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"sort"
	"strings"
	"syscall"
	"time"
)

const (
	cmdPerms = syscall.S_IRWXU | syscall.S_IRGRP | syscall.S_IXGRP |
		syscall.S_IROTH | syscall.S_IXOTH
	packagerPathname = "/bin/generic-packager"
)

var environmentToCopy = map[string]struct{}{
	"HOME":    struct{}{},
	"PATH":    struct{}{},
	"LOGNAME": struct{}{},
	"TZ":      struct{}{},
	"SHELL":   struct{}{},
	"USER":    struct{}{},
}

func (stream *bootstrapStream) build(b *Builder, client *srpc.Client,
	streamName string, expiresIn time.Duration, _ string, _ time.Duration,
	buildLog *bytes.Buffer, logger log.Logger) (string, error) {
	startTime := time.Now()
	args := make([]string, 0, len(stream.BootstrapCommand))
	rootDir, err := ioutil.TempDir("",
		strings.Replace(streamName, "/", "_", -1))
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(rootDir)
	for _, arg := range stream.BootstrapCommand {
		if arg == "$dir" {
			arg = rootDir
		}
		args = append(args, arg)
	}
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = buildLog
	cmd.Stderr = buildLog
	if err := cmd.Run(); err != nil {
		return "", err
	} else {
		packager := b.packagerTypes[stream.PackagerType]
		if err := packager.writePackageInstaller(rootDir); err != nil {
			return "", err
		}
		if err := clearResolvConf(buildLog, rootDir); err != nil {
			return "", err
		}
		buildDuration := time.Since(startTime)
		fmt.Fprintf(buildLog, "\nBuild time: %s\n",
			format.Duration(buildDuration))
		startTime = time.Now()
		name, err := addImage(client, streamName, rootDir, stream.Filter,
			nil, &filter.Filter{}, nil, expiresIn, buildLog, logger)
		if err != nil {
			return "", err
		}
		uploadDuration := time.Since(startTime)
		fmt.Fprintf(buildLog, "Built: %s in %s, uploaded in %s\n",
			name, format.Duration(buildDuration),
			format.Duration(uploadDuration))
		return name, nil
	}
}

func (packager *packagerType) writePackageInstaller(rootDir string) error {
	filename := path.Join(rootDir, packagerPathname)
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, cmdPerms)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	defer writer.Flush()
	fmt.Fprintln(writer, "#! /bin/sh")
	fmt.Fprintln(writer, "# Created by imaginator.")
	fmt.Fprintln(writer, "mount -n none -t proc /proc")
	fmt.Fprintln(writer, "mount -n none -t sysfs /sys")
	for _, line := range packager.Verbatim {
		fmt.Fprintln(writer, line)
	}
	fmt.Fprintln(writer, "cmd=\"$1\"; shift")
	writePackagerCommand(writer, "clean", packager.CleanCommand)
	fmt.Fprintln(writer, `[ "$cmd" = "copy-in" ] && exec cat > "$1"`)
	writePackagerCommand(writer, "install", packager.InstallCommand)
	fmt.Fprintln(writer, `[ "$cmd" = "run" ] && exec "$@"`)
	writePackagerCommand(writer, "update", packager.UpdateCommand)
	writePackagerCommand(writer, "upgrade", packager.UpgradeCommand)
	fmt.Fprintln(writer, "echo \"Invalid command: $cmd\"")
	fmt.Fprintln(writer, "exit 2")
	return writer.Flush()
}

func writePackagerCommand(writer io.Writer, cmd string, command []string) {
	if len(command) < 1 {
		fmt.Fprintf(writer, "[ \"$cmd\" = \"%s\" ] && exit 0\n", cmd)
	} else {
		fmt.Fprintf(writer, "[ \"$cmd\" = \"%s\" ] && exec %s \"$@\"\n",
			cmd, strings.Join(command, " "))
	}
}

func clearResolvConf(writer io.Writer, rootDir string) error {
	return runInTarget(nil, writer, rootDir, "truncate", "-s", "0",
		"/etc/resolv.conf")
}

func runInTarget(input io.Reader, output io.Writer, rootDir, prog string,
	args ...string) error {
	cmd := exec.Command(prog, args...)
	cmd.Env = stripVariables(os.Environ(), environmentToCopy)
	cmd.Dir = "/"
	cmd.Stdin = input
	cmd.Stdout = output
	cmd.Stderr = output
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Chroot:     rootDir,
		Setsid:     true,
		Cloneflags: syscall.CLONE_NEWNS | syscall.CLONE_NEWPID,
	}
	return cmd.Run()
}

func stripVariables(input []string, varsToCopy map[string]struct{}) []string {
	output := make([]string, 0)
	for _, nameValue := range os.Environ() {
		split := strings.SplitN(nameValue, "=", 2)
		if len(split) == 2 {
			if _, ok := varsToCopy[split[0]]; ok {
				output = append(output, nameValue)
			}
		}
	}
	sort.Strings(output)
	return output
}
