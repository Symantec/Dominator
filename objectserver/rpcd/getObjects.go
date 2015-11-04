package rpcd

import (
	"encoding/gob"
	"fmt"
	"github.com/Symantec/Dominator/proto/objectserver"
	"io"
	"net/http"
	"sync"
)

var exclusive sync.RWMutex

func getObjectsHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method != "CONNECT" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusMethodNotAllowed)
		io.WriteString(w, "405 must CONNECT\n")
		return
	}
	conn, bufrw, err := w.(http.Hijacker).Hijack()
	if err != nil {
		logger.Println("rpc hijacking ", req.RemoteAddr, ": ", err.Error())
		return
	}
	defer conn.Close()
	defer bufrw.Flush()
	io.WriteString(conn, "HTTP/1.0 200 Connected to GetObjects RPC\n\n")
	var request objectserver.GetObjectsRequest
	var response objectserver.GetObjectsResponse
	if request.Exclusive {
		exclusive.Lock()
		defer exclusive.Unlock()
	} else {
		exclusive.RLock()
		defer exclusive.RUnlock()
		getSemaphore <- true
		defer releaseSemaphore(getSemaphore)
	}
	decoder := gob.NewDecoder(bufrw)
	encoder := gob.NewEncoder(bufrw)
	if err = decoder.Decode(&request); err != nil {
		response.ResponseString = err.Error()
		encoder.Encode(response)
		return
	}
	response.ObjectSizes, err = objectServer.CheckObjects(request.Hashes)
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
	objectsReader, err := objectServer.GetObjects(request.Hashes)
	if err != nil {
		response.ResponseString = err.Error()
		encoder.Encode(response)
		return
	}
	encoder.Encode(response)
	bufrw.Flush()
	for _, hash := range request.Hashes {
		length, reader, err := objectsReader.NextObject()
		if err != nil {
			logger.Println(err)
			return
		}
		nCopied, err := io.Copy(conn, reader)
		reader.Close()
		if err != nil {
			logger.Printf("Error copying:\t%s\n", err)
			return
		}
		if nCopied != int64(length) {
			logger.Printf("Expected length: %d, got: %d for: %x\n",
				length, nCopied, hash)
		}
	}
}

func releaseSemaphore(semaphore chan bool) {
	<-semaphore
}
