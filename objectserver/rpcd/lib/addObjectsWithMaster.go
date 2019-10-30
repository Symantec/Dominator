package lib

import (
	"container/list"
	"encoding/gob"
	"fmt"
	"io"

	"github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/hash"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/objectserver"
	oclient "github.com/Cloud-Foundations/Dominator/lib/objectserver/client"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	proto "github.com/Cloud-Foundations/Dominator/proto/objectserver"
)

func addObjectsWithMaster(conn *srpc.Conn, decoder srpc.Decoder,
	encoder srpc.Encoder, objSrv objectserver.StashingObjectServer,
	masterAddress string, logger log.DebugLogger) error {
	logger.Printf("AddObjectsWithMaster(%s) starting\n", conn.RemoteAddr())
	defer conn.Flush()
	numAdded := 0
	numObj := 0
	outgoingQueueSendChan, outgoingQueueReceiveChan := newOutgoingQueue()
	errorChan := make(chan error, 1)
	defer close(outgoingQueueSendChan)
	go processOutgoingQueue(encoder, outgoingQueueReceiveChan, errorChan,
		logger)
	masterQueueSendChan, masterQueueReceiveChan := newMasterQueue()
	defer close(masterQueueSendChan)
	masterDrain := make(chan error, 1)
	masterClient, err := srpc.DialHTTP("tcp", masterAddress, 0)
	if err != nil {
		sendError(outgoingQueueSendChan, err)
		return nil
	}
	defer masterClient.Close()
	var masterConn *srpc.Conn
	var masterEncoder *gob.Encoder
	locallyKnownObjects := make([]hash.Hash, 0)
	// First process the stream of incoming objects, verify or stash+send.
	for ; ; numObj++ {
		if len(errorChan) > 0 {
			break
		}
		var request proto.AddObjectRequest
		var response proto.AddObjectResponse
		err := decoder.Decode(&request)
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}
			return errors.New("error decoding: " + err.Error())
		}
		if request.Length < 1 {
			break
		}
		var data []byte
		response.Hash, data, err = objSrv.StashOrVerifyObject(conn,
			request.Length, request.ExpectedHash)
		if err != nil {
			sendError(outgoingQueueSendChan, err)
			break
		}
		entry := make(chan proto.AddObjectResponse, 1)
		if data == nil { // Object already exists
			entry <- response
			outgoingQueueSendChan <- entry
			locallyKnownObjects = append(locallyKnownObjects, response.Hash)
			continue
		}
		if masterEncoder == nil {
			masterConn, err = masterClient.Call("ObjectServer.AddObjects")
			if err != nil {
				sendError(outgoingQueueSendChan,
					errors.New("error calling master: "+masterAddress+": "+
						err.Error()))
				break
			}
			defer func() {
				if masterConn != nil {
					masterConn.Close()
				}
			}()
			go processResponses(gob.NewDecoder(masterConn),
				masterQueueReceiveChan, masterDrain, objSrv)
			masterEncoder = gob.NewEncoder(masterConn)
			logger.Debugln(0,
				"AddObjectsWithMaster(): called ObjectServer.AddObjects for master")
		}
		err = sendRequest(masterConn, masterEncoder, data, response.Hash)
		if err != nil {
			sendError(outgoingQueueSendChan, err)
			break
		}
		masterQueueSendChan <- entry
		outgoingQueueSendChan <- entry
		numAdded++
	}
	logger.Debugln(0, "AddObjectsWithMaster(): Read all objects from client")
	if masterEncoder != nil {
		err := sendRequest(masterConn, masterEncoder, nil, hash.Hash{})
		if err != nil {
			sendError(outgoingQueueSendChan, err)
			logger.Printf(
				"AddObjectsWithMaster(): failed, %d of %d so far are new objects: %s",
				numAdded, numObj, err)
			return nil
		}
		masterQueueSendChan <- nil
		err = <-masterDrain
		masterConn.Close()
		masterConn = nil
		if err != nil {
			logger.Printf(
				"AddObjectsWithMaster(): failed, %d of %d so far are new objects: %s",
				numAdded, numObj, err)
			return nil
		}
	}
	// Check for objects missing on the master.
	err = sendMissingObjects(objSrv, masterClient, locallyKnownObjects, logger)
	if err != nil {
		sendError(outgoingQueueSendChan, err)
		err = <-errorChan
		logger.Printf(
			"AddObjectsWithMaster(): failed, %d of %d so far are new objects: %s",
			numAdded, numObj, err)
		return nil
	}
	logger.Debugln(0, "Sending flush request for outgoing queue")
	outgoingQueueSendChan <- nil
	logger.Debugln(0, "Sent flush request for outgoing queue")
	if err := <-errorChan; err != nil {
		logger.Printf(
			"AddObjectsWithMaster(): failed, %d of %d so far are new objects: %s",
			numAdded, numObj, err)
	} else {
		logger.Printf("AddObjectsWithMaster(): %d of %d are new objects",
			numAdded, numObj)
	}
	return nil
}

func processOutgoingQueue(encoder srpc.Encoder,
	queue <-chan <-chan proto.AddObjectResponse, errorChan chan<- error,
	logger log.DebugLogger) {
	// Hold back one response until flush entry. This ensures the client will
	// read an error message before seeing last "OK".
	var previousResponse *proto.AddObjectResponse
	for entry := range queue {
		var response proto.AddObjectResponse
		if entry != nil {
			response = <-entry
			if response.ErrorString != "" { // Preempt previous response.
				previousResponse = &response
			}
		}
		if previousResponse != nil {
			if err := encoder.Encode(*previousResponse); err != nil {
				err = errors.New("error encoding: " + err.Error())
				errorChan <- err
				break
			}
			if err := errors.New(previousResponse.ErrorString); err != nil {
				errorChan <- err
				break
			}
		}
		if entry == nil {
			errorChan <- nil
			break
		}
		previousResponse = &response
	}
	logger.Debugln(0, "Draining outgoing queue")
	for range queue { // Drain the queue.
	}
}

func processResponses(decoder *gob.Decoder,
	queue <-chan chan<- proto.AddObjectResponse, completed chan<- error,
	objSrv objectserver.StashingObjectServer) {
	var err error
	for entry := range queue {
		if entry == nil {
			break
		}
		var reply proto.AddObjectResponse
		if err = decoder.Decode(&reply); err != nil {
			reply.ErrorString = "error decoding: " + err.Error()
			entry <- reply
			break
		}
		if reply.ErrorString == "" {
			if err = objSrv.CommitObject(reply.Hash); err != nil {
				reply.ErrorString = "commit error: " + err.Error()
				entry <- reply
				break
			}
		}
		entry <- reply
	}
	completed <- err
	for range queue { // Drain the queue.
	}
}

func sendRequest(conn *srpc.Conn, encoder *gob.Encoder, data []byte,
	hashVal hash.Hash) error {
	request := proto.AddObjectRequest{
		Length:       uint64(len(data)),
		ExpectedHash: &hashVal,
	}
	if err := encoder.Encode(request); err != nil {
		return err
	}
	if len(data) > 0 {
		_, err := conn.Write(data)
		return err
	}
	return conn.Flush()
}

func sendError(queue chan<- <-chan proto.AddObjectResponse, err error) {
	entry := make(chan proto.AddObjectResponse, 1)
	entry <- proto.AddObjectResponse{ErrorString: err.Error()}
	queue <- entry
}

func sendMissingObjects(objSrv objectserver.StashingObjectServer,
	masterClient *srpc.Client, locallyKnownObjects []hash.Hash,
	logger log.DebugLogger) error {
	if len(locallyKnownObjects) < 1 {
		return nil
	}
	logger.Debugf(0,
		"Checking %d locally known objects for presence on master\n",
		len(locallyKnownObjects))
	objClient := oclient.AttachObjectClient(masterClient)
	lengths, err := objClient.CheckObjects(locallyKnownObjects)
	objClient.Close()
	if err != nil {
		return fmt.Errorf("error checking master for known objects: %s", err)
	}
	allObjectsKnownOnMaster := true
	for _, lengthOnMaster := range lengths {
		if lengthOnMaster < 1 {
			allObjectsKnownOnMaster = false
			break
		}
	}
	if allObjectsKnownOnMaster {
		return nil
	}
	adderQueue, err := oclient.NewObjectAdderQueue(masterClient)
	if err != nil {
		return err
	}
	defer func() {
		if adderQueue != nil {
			adderQueue.Close()
		}
	}()
	numSent := 0
	for index, lengthOnMaster := range lengths {
		if lengthOnMaster > 0 {
			continue
		}
		hashVal := locallyKnownObjects[index]
		length, reader, err := objectserver.GetObject(objSrv, hashVal)
		if err != nil {
			return err
		}
		data := make([]byte, length)
		nRead, err := io.ReadFull(reader, data)
		reader.Close()
		if err != nil {
			return err
		}
		if uint64(nRead) != length {
			return fmt.Errorf(
				"failed to read file data, wanted: %d, got: %d bytes",
				length, nRead)
		}
		if err := adderQueue.AddData(data, hashVal); err != nil {
			return err
		}
		numSent++
	}
	err = adderQueue.Close()
	adderQueue = nil
	logger.Printf("AddObjectsWithMaster() Sent: %d of %d locally known objects\n",
		numSent, len(locallyKnownObjects))
	return err
}

func newOutgoingQueue() (chan<- <-chan proto.AddObjectResponse,
	<-chan <-chan proto.AddObjectResponse) {
	send := make(chan (<-chan proto.AddObjectResponse), 1)
	receive := make(chan (<-chan proto.AddObjectResponse), 1)
	go manageOutgoingQueue(send, receive)
	return send, receive
}

func manageOutgoingQueue(send <-chan <-chan proto.AddObjectResponse,
	receive chan<- <-chan proto.AddObjectResponse) {
	queue := list.New()
	for {
		if front := queue.Front(); front == nil {
			if send == nil {
				close(receive)
				return
			}
			response, ok := <-send
			if !ok {
				close(receive)
				return
			}
			queue.PushBack(response)
		} else {
			select {
			case receive <- front.Value.(<-chan proto.AddObjectResponse):
				queue.Remove(front)
			case response, ok := <-send:
				if ok {
					queue.PushBack(response)
				} else {
					send = nil
				}
			}
		}
	}
}

func newMasterQueue() (chan<- chan<- proto.AddObjectResponse,
	<-chan chan<- proto.AddObjectResponse) {
	send := make(chan chan<- proto.AddObjectResponse, 1)
	receive := make(chan chan<- proto.AddObjectResponse, 1)
	go manageMasterQueue(send, receive)
	return send, receive
}

func manageMasterQueue(send <-chan chan<- proto.AddObjectResponse,
	receive chan<- chan<- proto.AddObjectResponse) {
	queue := list.New()
	for {
		if front := queue.Front(); front == nil {
			if send == nil {
				close(receive)
				return
			}
			response, ok := <-send
			if !ok {
				close(receive)
				return
			}
			queue.PushBack(response)
		} else {
			select {
			case receive <- front.Value.(chan<- proto.AddObjectResponse):
				queue.Remove(front)
			case response, ok := <-send:
				if ok {
					queue.PushBack(response)
				} else {
					send = nil
				}
			}
		}
	}
}
