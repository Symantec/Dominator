package main

import (
	"fmt"
	"os"
	"time"

	"github.com/Symantec/Dominator/lib/log"
	terminalclient "github.com/Symantec/Dominator/lib/net/terminal/client"
	"github.com/Symantec/Dominator/lib/srpc"
)

func installerShellSubcommand(args []string, logger log.DebugLogger) error {
	err := installerShell(args[0], logger)
	if err != nil {
		return fmt.Errorf("Error talking to installer shell: %s", err)
	}
	return nil
}

func installerShell(hostname string, logger log.DebugLogger) error {
	var client *srpc.Client
	fmt.Fprintf(os.Stderr, "trying to connect")
	for ; ; time.Sleep(time.Second * 5) {
		var err error
		client, err = srpc.DialHTTP("tcp", fmt.Sprintf("%s:%d",
			hostname, *installerPortNum), time.Second*15)
		if err == nil {
			break
		}
		fmt.Fprintf(os.Stderr, ".")
	}
	defer client.Close()
	conn, err := client.Call("Installer.Shell")
	if err != nil {
		return err
	}
	defer conn.Close()
	fmt.Fprintf(os.Stderr, " connected...\n")
	if err := terminalclient.StartTerminal(conn); err != nil {
		return err
	}
	fmt.Fprint(os.Stderr, "\r")
	return nil
}
