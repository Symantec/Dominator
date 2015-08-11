package rpcd

import (
	"github.com/Symantec/Dominator/proto/objectserver"
)

func (t *rpcType) AddFiles(request objectserver.AddFilesRequest,
	reply *objectserver.AddFilesResponse) error {
	var response objectserver.AddFilesResponse
	for _, objectToAdd := range request.ObjectsToAdd {
		hash, err := objectServer.PutObject(objectToAdd.ObjectData,
			objectToAdd.ExpectedHash)
		if err != nil {
			return err
		}
		response.Hashes = append(response.Hashes, hash)
	}
	*reply = response
	return nil
}
