package rpcd

import (
	"bufio"
	"encoding/gob"
	"fmt"
	"github.com/Symantec/Dominator/proto/objectserver"
	"io"
	"net"
	"sync"
)

var exclusive sync.RWMutex

func (objSrv *srpcType) GetObjects(conn net.Conn) {
	bufrw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
	defer bufrw.Flush()
	objSrv.getObjects(bufrw)
}

func (objSrv *srpcType) getObjects(bufrw *bufio.ReadWriter) {
	var request objectserver.GetObjectsRequest
	var response objectserver.GetObjectsResponse
	if request.Exclusive {
		exclusive.Lock()
		defer exclusive.Unlock()
	} else {
		exclusive.RLock()
		defer exclusive.RUnlock()
		objSrv.getSemaphore <- true
		defer releaseSemaphore(objSrv.getSemaphore)
	}
	decoder := gob.NewDecoder(bufrw)
	encoder := gob.NewEncoder(bufrw)
	var err error
	if err = decoder.Decode(&request); err != nil {
		response.ResponseString = err.Error()
		encoder.Encode(response)
		return
	}
	response.ObjectSizes, err = objSrv.objectServer.CheckObjects(request.Hashes)
	if err != nil {
		response.ResponseString = err.Error()
		encoder.Encode(response)
		return
	}
	// First a quick check for existence. If any objects missing, fail request.
	for index, hash := range request.Hashes {
		if response.ObjectSizes[index] < 1 {
			response.ResponseString = fmt.Sprintf("unknown object: %x", hash)
			encoder.Encode(response)
			return
		}
	}
	objectsReader, err := objSrv.objectServer.GetObjects(request.Hashes)
	if err != nil {
		response.ResponseString = err.Error()
		encoder.Encode(response)
		return
	}
	defer objectsReader.Close()
	encoder.Encode(response)
	bufrw.Flush()
	for _, hash := range request.Hashes {
		length, reader, err := objectsReader.NextObject()
		if err != nil {
			objSrv.logger.Println(err)
			return
		}
		nCopied, err := io.Copy(bufrw.Writer, reader)
		reader.Close()
		if err != nil {
			objSrv.logger.Printf("Error copying:\t%s\n", err)
			return
		}
		if nCopied != int64(length) {
			objSrv.logger.Printf("Expected length: %d, got: %d for: %x\n",
				length, nCopied, hash)
			return
		}
	}
	objSrv.logger.Printf("GetObjects() sent: %d objects\n", len(request.Hashes))
}

func releaseSemaphore(semaphore <-chan bool) {
	<-semaphore
}
