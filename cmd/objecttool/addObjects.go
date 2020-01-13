package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/Cloud-Foundations/Dominator/lib/log"
	objectclient "github.com/Cloud-Foundations/Dominator/lib/objectserver/client"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
)

func addObjectsSubcommand(args []string, logger log.DebugLogger) error {
	if err := addObjects(fmt.Sprintf("%s:%d",
		*objectServerHostname, *objectServerPortNum), args); err != nil {
		return fmt.Errorf("Error adding objects hash: %s", err)
	}
	return nil
}

func addObjects(address string, filenames []string) error {
	client, err := srpc.DialHTTP("tcp", address, 0)
	if err != nil {
		return err
	}
	defer client.Close()
	objQ, err := objectclient.NewObjectAdderQueue(client)
	if err != nil {
		return err
	}
	for _, filename := range filenames {
		if err := addObject(objQ, filename); err != nil {
			objQ.Close()
			return err
		}
	}
	return objQ.Close()
}

func addObject(objQ *objectclient.ObjectAdderQueue, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	fi, err := file.Stat()
	if err != nil {
		return err
	}
	if !fi.Mode().IsRegular() {
		return nil
	}
	_, err = objQ.Add(bufio.NewReader(file), uint64(fi.Size()))
	return err
}
