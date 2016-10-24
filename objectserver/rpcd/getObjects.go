package rpcd

import (
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/objectserver"
	"io"
	"sync"
)

var exclusive sync.RWMutex

func (objSrv *srpcType) GetObjects(conn *srpc.Conn) error {
	defer conn.Flush()
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
	decoder := gob.NewDecoder(conn)
	encoder := gob.NewEncoder(conn)
	var err error
	if err = decoder.Decode(&request); err != nil {
		response.ResponseString = err.Error()
		return encoder.Encode(response)
	}
	response.ObjectSizes, err = objSrv.objectServer.CheckObjects(request.Hashes)
	if err != nil {
		response.ResponseString = err.Error()
		return encoder.Encode(response)
	}
	// First a quick check for existence. If any objects missing, fail request.
	for index, hash := range request.Hashes {
		if response.ObjectSizes[index] < 1 {
			response.ResponseString = fmt.Sprintf("unknown object: %x", hash)
			return encoder.Encode(response)
		}
	}
	objectsReader, err := objSrv.objectServer.GetObjects(request.Hashes)
	if err != nil {
		response.ResponseString = err.Error()
		return encoder.Encode(response)
	}
	defer objectsReader.Close()
	if err := encoder.Encode(response); err != nil {
		return err
	}
	conn.Flush()
	for _, hash := range request.Hashes {
		length, reader, err := objectsReader.NextObject()
		if err != nil {
			objSrv.logger.Println(err)
			return err
		}
		nCopied, err := io.Copy(conn.Writer, reader)
		reader.Close()
		if err != nil {
			objSrv.logger.Printf("Error copying: %s\n", err)
			return err
		}
		if nCopied != int64(length) {
			txt := fmt.Sprintf("Expected length: %d, got: %d for: %x",
				length, nCopied, hash)
			objSrv.logger.Printf(txt)
			return errors.New(txt)
		}
	}
	objSrv.logger.Printf("GetObjects() sent: %d objects\n", len(request.Hashes))
	return nil
}

func releaseSemaphore(semaphore <-chan bool) {
	<-semaphore
}
