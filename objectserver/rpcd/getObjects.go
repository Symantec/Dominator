package rpcd

import (
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/proto/objectserver"
	"io/ioutil"
	"runtime"
)

func (t *rpcType) GetObjects(request objectserver.GetObjectsRequest,
	reply *objectserver.GetObjectsResponse) error {
	var response objectserver.GetObjectsResponse
	// First a quick check for existence. If any objects missing, fail request.
	objectsPresent, err := objectServer.CheckObjects(request.Hashes)
	if err != nil {
		return err
	}
	for index, hash := range request.Hashes {
		if !objectsPresent[index] {
			return errors.New(fmt.Sprintf("unknown object: %x", hash))
		}
	}
	response.ObjectSizes = make([]uint64, len(request.Hashes))
	response.Objects = make([][]byte, len(request.Hashes))
	objectsReader, err := objectServer.GetObjects(request.Hashes)
	if err != nil {
		return err
	}
	for index := range request.Hashes {
		size, reader, err := objectsReader.NextObject()
		if err != nil {
			return err
		}
		response.ObjectSizes[index] = size
		response.Objects[index], err = ioutil.ReadAll(reader)
		reader.Close()
		if err != nil {
			return errors.New(fmt.Sprintf(
				"error reading data for object: %s %s", err.Error()))
		}
	}
	*reply = response
	runtime.GC() // An opportune time to take out the garbage.
	return nil
}
