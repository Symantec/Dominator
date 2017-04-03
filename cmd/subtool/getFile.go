package main

import (
	"bufio"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/sub/client"
	"io"
	"os"
)

func getFileSubcommand(getSubClient getSubClientFunc, args []string) {
	if err := getFile(getSubClient(), args[0], args[1]); err != nil {
		logger.Fatalf("Error getting file: %s\n", err)
	}
	os.Exit(0)
}

func getFile(srpcClient *srpc.Client, remoteFile, localFile string) error {
	readerFunc := func(reader io.Reader, size uint64) error {
		file, err := os.Create(localFile)
		if err != nil {
			return err
		}
		defer file.Close()
		writer := bufio.NewWriter(file)
		defer writer.Flush()
		_, err = io.Copy(writer, &io.LimitedReader{R: reader, N: int64(size)})
		return err
	}
	rfiles := make([]string, 1)
	rfiles[0] = remoteFile
	return client.GetFiles(srpcClient, rfiles, readerFunc)
}
