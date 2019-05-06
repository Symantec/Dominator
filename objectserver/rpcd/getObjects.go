package rpcd

import (
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/objectserver"
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
	var err error
	if err = conn.Decode(&request); err != nil {
		response.ResponseString = err.Error()
		return conn.Encode(response)
	}
	response.ObjectSizes, err = objSrv.objectServer.CheckObjects(request.Hashes)
	if err != nil {
		response.ResponseString = err.Error()
		return conn.Encode(response)
	}
	// First a quick check for existence. If any objects missing, fail request.
	var firstMissingObject *hash.Hash
	numMissingObjects := 0
	for index, hashVal := range request.Hashes {
		if response.ObjectSizes[index] < 1 {
			firstMissingObject = &hashVal
			numMissingObjects++
		}
	}
	if firstMissingObject != nil {
		if numMissingObjects == 1 {
			response.ResponseString = fmt.Sprintf("unknown object: %x",
				*firstMissingObject)
		} else {
			response.ResponseString = fmt.Sprintf(
				"first of %d unknown objects: %x", numMissingObjects,
				*firstMissingObject)
		}
		return conn.Encode(response)
	}
	objectsReader, err := objSrv.objectServer.GetObjects(request.Hashes)
	if err != nil {
		response.ResponseString = err.Error()
		return conn.Encode(response)
	}
	defer objectsReader.Close()
	if err := conn.Encode(response); err != nil {
		return err
	}
	conn.Flush()
	buffer := make([]byte, 32<<10)
	for _, hashVal := range request.Hashes {
		length, reader, err := objectsReader.NextObject()
		if err != nil {
			objSrv.logger.Println(err)
			return err
		}
		nCopied, err := io.CopyBuffer(conn, reader, buffer)
		reader.Close()
		if err != nil {
			objSrv.logger.Printf("Error copying: %s\n", err)
			return err
		}
		if nCopied != int64(length) {
			txt := fmt.Sprintf("Expected length: %d, got: %d for: %x",
				length, nCopied, hashVal)
			objSrv.logger.Printf(txt)
			return errors.New(txt)
		}
	}
	objSrv.logger.Debugf(0, "GetObjects() sent: %d objects\n",
		len(request.Hashes))
	return nil
}

func releaseSemaphore(semaphore <-chan bool) {
	<-semaphore
}
