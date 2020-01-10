package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Cloud-Foundations/Dominator/lib/constants"
	"github.com/Cloud-Foundations/Dominator/lib/flags/loadflags"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/lib/srpc/setupclient"
)

var (
	imageUnpackerHostname = flag.String("imageUnpackerHostname", "localhost",
		"Hostname of image-unpacker server")
	imageUnpackerPortNum = flag.Uint("imageUnpackerPortNum",
		constants.ImageUnpackerPortNumber,
		"Port number of image-unpacker server")
)

func printSubcommands(subcommands []subcommand) {
	for _, subcommand := range subcommands {
		if subcommand.args == "" {
			fmt.Fprintln(os.Stderr, " ", subcommand.command)
		} else {
			fmt.Fprintln(os.Stderr, " ", subcommand.command, subcommand.args)
		}
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr,
		"Usage: unpacker-tool [flags...] add-device [args...]")
	fmt.Fprintln(os.Stderr, "Common flags:")
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr, "Commands:")
	printSubcommands(subcommands)
}

type commandFunc func(*srpc.Client, []string) error

type subcommand struct {
	command string
	args    string
	minArgs int
	maxArgs int
	cmdFunc commandFunc
}

var subcommands = []subcommand{
	{"add-device", "DeviceId command ...", 2, -1, addDeviceSubcommand},
	{"associate", "stream-name DeviceId", 2, 2, associateSubcommand},
	{"export-image", "stream-name type destination", 3, 3,
		exportImageSubcommand},
	{"get-status", "", 0, 0, getStatusSubcommand},
	{"get-device-for-stream", "stream-name", 1, 1,
		getDeviceForStreamSubcommand},
	{"prepare-for-capture", "stream-name", 1, 1, prepareForCaptureSubcommand},
	{"prepare-for-copy", "stream-name", 1, 1, prepareForCopySubcommand},
	{"prepare-for-unpack", "stream-name", 1, 1, prepareForUnpackSubcommand},
	{"remove-device", "DeviceId", 1, 1, removeDeviceSubcommand},
	{"unpack-image", "stream-name image-leaf-name", 2, 2,
		unpackImageSubcommand},
}

func doMain() int {
	if err := loadflags.LoadForCli("unpacker-tool"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	flag.Usage = printUsage
	flag.Parse()
	if flag.NArg() < 1 {
		printUsage()
		return 2
	}
	if err := setupclient.SetupTls(true); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	clientName := fmt.Sprintf("%s:%d",
		*imageUnpackerHostname, *imageUnpackerPortNum)
	client, err := srpc.DialHTTP("tcp", clientName, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error dialing\t%s\n", err)
		return 1
	}
	numSubcommandArgs := flag.NArg() - 1
	for _, subcommand := range subcommands {
		if flag.Arg(0) == subcommand.command {
			if numSubcommandArgs < subcommand.minArgs ||
				(subcommand.maxArgs >= 0 &&
					numSubcommandArgs > subcommand.maxArgs) {
				printUsage()
				return 2
			}
			if err := subcommand.cmdFunc(client, flag.Args()[1:]); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
			return 0
		}
	}
	printUsage()
	return 2
}

func main() {
	os.Exit(doMain())
}
