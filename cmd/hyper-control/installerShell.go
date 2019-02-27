package main

import (
	"fmt"
	"os"
	"time"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
)

func installerShellSubcommand(args []string, logger log.DebugLogger) {
	err := installerShell(args[0], logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error talking to installer shell: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
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
	closed := false
	defer func() {
		closed = true
	}()
	fmt.Fprintf(os.Stderr, " connected...\n")
	oldState, err := terminal.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}
	defer func() {
		terminal.Restore(int(os.Stdin.Fd()), oldState)
		fmt.Println()
	}()
	go func() { // Read from connection, write to stdout.
		for {
			buffer := make([]byte, 256)
			if nRead, err := conn.Read(buffer); err != nil {
				if closed {
					return
				}
				logger.Printf("error reading from remote shell: %s\n", err)
				return
			} else {
				os.Stderr.Write(buffer[:nRead])
			}
		}
	}()
	// Read from stdin until control-d (EOT).
	for {
		buffer := make([]byte, 256)
		if nRead, err := os.Stdin.Read(buffer); err != nil {
			return fmt.Errorf("error reading from stdin: %s", err)
		} else {
			if buffer[0] == '\x04' { // Control-d: EndOfTransmission.
				return nil
			}
			if _, err := conn.Write(buffer[:nRead]); err != nil {
				return fmt.Errorf("error writing to connection: %s", err)
			}
			if err := conn.Flush(); err != nil {
				return fmt.Errorf("error flushing connection: %s", err)
			}
		}
	}
}
