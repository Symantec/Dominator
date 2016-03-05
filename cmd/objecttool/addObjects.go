package main

import (
	"bufio"
	"fmt"
	"github.com/Symantec/Dominator/lib/objectclient"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/objectserver"
	"os"
)

func addObjectsSubcommand(objSrv objectserver.ObjectServer, args []string) {
	if err := addObjects(fmt.Sprintf("%s:%d",
		*objectServerHostname, *objectServerPortNum), args); err != nil {
		fmt.Fprintf(os.Stderr, "Error adding obnects hash\t%s\n", err)
		os.Exit(2)
	}
	os.Exit(0)
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
