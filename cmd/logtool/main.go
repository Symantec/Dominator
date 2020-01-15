package main

import (
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/Cloud-Foundations/Dominator/lib/flags/commands"
	"github.com/Cloud-Foundations/Dominator/lib/flags/loadflags"
	"github.com/Cloud-Foundations/Dominator/lib/log/cmdlogger"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/lib/srpc/setupclient"
)

var (
	excludeRegex = flag.String("excludeRegex", "",
		"The exclude regular expression to filter out when watching (after include)")
	includeRegex = flag.String("includeRegex", "",
		"The include regular expression to filter for when watching")
	loggerHostname = flag.String("loggerHostname", "localhost",
		"Hostname of log server")
	loggerName    = flag.String("loggerName", "", "Name of logger")
	loggerPortNum = flag.Uint("loggerPortNum", 0, "Port number of log server")
)

func printUsage() {
	w := flag.CommandLine.Output()
	fmt.Fprintln(w,
		"Usage: logtool [flags...] debug|print|set-debug-level [args...]")
	fmt.Fprintln(w, "Common flags:")
	flag.PrintDefaults()
	fmt.Fprintln(w, "Commands:")
	commands.PrintCommands(w, subcommands)
}

var subcommands = []commands.Command{
	{"debug", "          level args...", 2, -1, debugSubcommand},
	{"print", "                args...", 1, -1, printSubcommand},
	{"set-debug-level", "level", 1, 1, setDebugLevelSubcommand},
	{"watch", "          level", 1, 1, watchSubcommand},
}

func dial(allowMultiClient bool) ([]*srpc.Client, []string, error) {
	if addrs, err := net.LookupHost(*loggerHostname); err != nil {
		return nil, nil, err
	} else if len(addrs) < 1 {
		return nil, nil, fmt.Errorf("no addresses for: %s", *loggerHostname)
	} else if !allowMultiClient && len(addrs) > 1 {
		return nil, nil, fmt.Errorf("multiple endpoints not supported")
	} else {
		clients, err := dialAll(addrs)
		return clients, addrs, err
	}
}

func dialAll(addrs []string) ([]*srpc.Client, error) {
	clients := make([]*srpc.Client, 0, len(addrs))
	for _, addr := range addrs {
		clientName := fmt.Sprintf("%s:%d", addr, *loggerPortNum)
		if client, err := srpc.DialHTTP("tcp", clientName, 0); err != nil {
			return nil, err
		} else {
			clients = append(clients, client)
		}
	}
	return clients, nil
}

func doMain() int {
	if err := loadflags.LoadForCli("logtool"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	flag.Usage = printUsage
	flag.Parse()
	if flag.NArg() < 1 {
		printUsage()
		return 2
	}
	logger := cmdlogger.New()
	if err := setupclient.SetupTls(true); err != nil {
		logger.Fatalln(err)
	}
	return commands.RunCommands(subcommands, printUsage, logger)
}

func main() {
	os.Exit(doMain())
}
