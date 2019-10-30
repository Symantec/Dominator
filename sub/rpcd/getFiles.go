package rpcd

import (
	"bufio"
	"io"
	"os"
	"path"

	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/sub"
)

func (t *rpcType) GetFiles(conn *srpc.Conn) error {
	defer conn.Flush()
	t.getFilesLock.Lock()
	defer t.getFilesLock.Unlock()
	numFiles := 0
	for ; ; numFiles++ {
		filename, err := conn.ReadString('\n')
		if err != nil {
			return err
		}
		filename = filename[:len(filename)-1]
		if filename == "" {
			break
		}
		filename = path.Join(t.rootDir, filename)
		if err := processFilename(conn, filename); err != nil {
			return err
		}
	}
	plural := "s"
	if numFiles == 1 {
		plural = ""
	}
	t.logger.Printf("GetFiles(): %d file%s provided\n", numFiles, plural)
	return nil
}

func processFilename(conn *srpc.Conn, filename string) error {
	file, err := os.Open(filename)
	var response sub.GetFileResponse
	if err != nil {
		response.Error = err
	} else {
		defer file.Close()
		if fi, err := file.Stat(); err != nil {
			response.Error = err
		} else {
			response.Size = uint64(fi.Size())
		}
	}
	if err := conn.Encode(response); err != nil {
		return err
	}
	if response.Error != nil {
		return nil
	}
	_, err = io.Copy(conn, bufio.NewReader(file))
	return err
}
