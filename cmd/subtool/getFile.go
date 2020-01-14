package main

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/sub/client"
)

func getFileSubcommand(args []string, logger log.DebugLogger) error {
	srpcClient := getSubClient(logger)
	defer srpcClient.Close()
	if err := getFile(srpcClient, args[0], args[1]); err != nil {
		return fmt.Errorf("Error getting file: %s", err)
	}
	return nil
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
