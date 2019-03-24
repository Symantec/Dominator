package client

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/ssh/terminal"
)

type flushWriter interface {
	Flush() error
	io.Writer
}

func startTerminal(conn FlushReadWriter) error {
	closed := false
	defer func() {
		closed = true
	}()
	oldState, err := terminal.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}
	defer terminal.Restore(int(os.Stdin.Fd()), oldState)
	retval := make(chan error, 2)
	go func(retval chan<- error) {
		retval <- readFromConnection(conn)
	}(retval)
	go func(retval chan<- error) {
		retval <- readFromStdin(conn)
	}(retval)
	return <-retval
}

func readFromConnection(conn io.Reader) error {
	// Read from connection until EOF, write to stdout.
	buffer := make([]byte, 256)
	for {
		if nRead, err := conn.Read(buffer); err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("error reading from remote shell: %s\n", err)
		} else {
			os.Stderr.Write(buffer[:nRead])
		}
	}
}

func readFromStdin(conn flushWriter) error {
	// Read from stdin until control-\ (File separator), write to connection.
	for {
		buffer := make([]byte, 256)
		if nRead, err := os.Stdin.Read(buffer); err != nil {
			return fmt.Errorf("error reading from stdin: %s", err)
		} else {
			if buffer[0] == '\x1c' { // Control-\: File separator.
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
