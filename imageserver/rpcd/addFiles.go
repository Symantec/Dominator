package rpcd

import (
	"github.com/Symantec/Dominator/proto/imageserver"
)

func (t *ImageServer) AddFiles(request imageserver.AddFilesRequest,
	reply *imageserver.AddFilesResponse) error {
	var response imageserver.AddFilesResponse
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
