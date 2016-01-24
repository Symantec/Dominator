package objectclient

import (
	"crypto/sha512"
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/objectserver"
	"io"
)

func newObjectAdderQueue(objClient *ObjectClient) (*ObjectAdderQueue, error) {
	var objQ ObjectAdderQueue
	var err error
	objQ.client, err = srpc.DialHTTP("tcp", objClient.address, 0)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Error dialing\t%s\n", err.Error()))
	}
	objQ.conn, err = objQ.client.Call("ObjectServer.AddObjects")
	if err != nil {
		objQ.client.Close()
		return nil, err
	}
	objQ.encoder = gob.NewEncoder(objQ.conn)
	getResponseChan := make(chan bool, 65536)
	errorChan := make(chan error, 1024)
	objQ.getResponseChan = getResponseChan
	objQ.errorChan = errorChan
	objQ.sendSemaphore = make(chan bool, 1)
	go readResponses(objQ.conn, getResponseChan, errorChan)
	return &objQ, nil
}

func (objQ *ObjectAdderQueue) add(reader io.Reader, length uint64) (
	hash.Hash, error) {
	var hash hash.Hash
	if err := objQ.consumeErrors(false); err != nil {
		return hash, err
	}
	data := make([]byte, length)
	nRead, err := io.ReadFull(reader, data)
	if err != nil {
		return hash, err
	}
	if uint64(nRead) != length {
		return hash, errors.New(fmt.Sprintf(
			"failed to read file data, wanted: %d, got: %d bytes",
			length, nRead))
	}
	hasher := sha512.New()
	if _, err := hasher.Write(data); err != nil {
		return hash, err
	}
	copy(hash[:], hasher.Sum(nil))
	// Send in a goroutine to increase concurrency. A small win.
	objQ.sendSemaphore <- true
	go func() {
		defer func() {
			<-objQ.sendSemaphore
		}()
		var request objectserver.AddObjectRequest
		request.Length = uint64(len(data))
		request.ExpectedHash = &hash
		objQ.encoder.Encode(request)
		objQ.conn.Write(data)
		objQ.getResponseChan <- true
	}()
	return hash, nil
}

func (objQ *ObjectAdderQueue) close() error {
	objQ.sendSemaphore <- true // Wait for any sends in progress to complete.
	var request objectserver.AddObjectRequest
	err := objQ.encoder.Encode(request)
	err = updateError(err, objQ.conn.Flush())
	close(objQ.getResponseChan)
	err = updateError(err, objQ.consumeErrors(true))
	err = updateError(err, objQ.conn.Close())
	return updateError(err, objQ.client.Close())
}

func updateError(oldError, newError error) error {
	if oldError == nil {
		return newError
	}
	return oldError
}

func (objQ *ObjectAdderQueue) consumeErrors(untilClose bool) error {
	if untilClose {
		for err := range objQ.errorChan {
			if err != nil {
				return err
			}
		}
	} else {
		for len(objQ.errorChan) > 0 {
			err := <-objQ.errorChan
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func readResponses(conn *srpc.Conn, getResponseChan <-chan bool,
	errorChan chan<- error) {
	defer close(errorChan)
	decoder := gob.NewDecoder(conn)
	for range getResponseChan {
		var reply objectserver.AddObjectResponse
		err := decoder.Decode(&reply)
		if err == nil {
			err = reply.Error
		}
		errorChan <- err
		if err != nil {
			break
		}
	}
}
