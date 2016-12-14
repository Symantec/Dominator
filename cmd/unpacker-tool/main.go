package main

import (
	"flag"
	"fmt"
	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/lib/srpc/setupclient"
	"os"
)

var (
	imageUnpackerHostname = flag.String("imageUnpackerHostname", "localhost",
		"Hostname of image-unpacker server")
	imageUnpackerPortNum = flag.Uint("imageUnpackerPortNum",
		constants.ImageUnpackerPortNumber,
		"Port number of image-unpacker server")
)

func printUsage() {
	fmt.Fprintln(os.Stderr,
		"Usage: unpacker-tool [flags...] add-device [args...]")
	fmt.Fprintln(os.Stderr, "Common flags:")
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  add-device DeviceId command ...")
	fmt.Fprintln(os.Stderr, "  associate stream-name DeviceId")
	fmt.Fprintln(os.Stderr, "  get-status")
	fmt.Fprintln(os.Stderr, "  get-device-for-stream stream-name")
	fmt.Fprintln(os.Stderr, "  prepare-for-capture stream-name")
	fmt.Fprintln(os.Stderr, "  prepare-for-unpack stream-name")
	fmt.Fprintln(os.Stderr, "  unpack-image image-name")
}

type commandFunc func(*srpc.Client, []string)

type subcommand struct {
	command string
	minArgs int
	maxArgs int
	cmdFunc commandFunc
}

var subcommands = []subcommand{
	{"add-device", 2, -1, addDeviceSubcommand},
	{"associate", 2, 2, associateSubcommand},
	{"get-status", 0, 0, getStatusSubcommand},
	{"get-device-for-stream", 1, 1, getDeviceForStreamSubcommand},
	{"prepare-for-capture", 1, 1, prepareForCaptureSubcommand},
	{"prepare-for-unpack", 1, 1, prepareForUnpackSubcommand},
	{"unpack-image", 1, 1, unpackImageSubcommand},
}

func main() {
	flag.Usage = printUsage
	flag.Parse()
	if flag.NArg() < 1 {
		printUsage()
		os.Exit(2)
	}
	if err := setupclient.SetupTls(true); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	clientName := fmt.Sprintf("%s:%d",
		*imageUnpackerHostname, *imageUnpackerPortNum)
	client, err := srpc.DialHTTP("tcp", clientName, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error dialing\t%s\n", err)
		os.Exit(1)
	}

	numSubcommandArgs := flag.NArg() - 1
	for _, subcommand := range subcommands {
		if flag.Arg(0) == subcommand.command {
			if numSubcommandArgs < subcommand.minArgs ||
				(subcommand.maxArgs >= 0 &&
					numSubcommandArgs > subcommand.maxArgs) {
				printUsage()
				os.Exit(2)
			}
			subcommand.cmdFunc(client, flag.Args()[1:])
			os.Exit(3)
		}
	}
	printUsage()
	os.Exit(2)
}
