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
	for _, hash := range request.Objects {
		found, err := objectServer.CheckObject(hash)
		if err != nil {
			return err
		}
		if !found {
			return errors.New(fmt.Sprintf("unknown object: %x", hash))
		}
	}
	response.ObjectSizes = make([]uint64, len(request.Objects))
	response.Objects = make([][]byte, len(request.Objects))
	for index, hash := range request.Objects {
		size, reader, err := objectServer.GetObjectReader(hash)
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
